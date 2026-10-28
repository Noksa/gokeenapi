package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Exporter serves cached snapshots collected by independent router pollers.
type Exporter struct {
	cfg     config.ExporterConfig
	targets []*target
	slots   chan struct{}
	handler http.Handler
}

type target struct {
	name       string
	interfaces []string
	session    *gokeenrestapi.MonitoringSession
	pollMu     sync.Mutex
	mu         sync.RWMutex
	collectors []*collectorState
}

type collectorState struct {
	name        string
	settings    config.ExporterCollector
	next        time.Time
	attempted   time.Time
	lastSuccess time.Time
	duration    time.Duration
	success     bool
	failures    uint
	errors      uint64
	samples     []sample
}

var collectorPaths = map[string]string{
	"system": "/rci/show/system", "dns": "/rci/show/dns-proxy", "interfaces": "/rci/show/interface", "hosts": "/rci/show/ip/hotspot",
	"static_routes": "/rci/ip/route", "dns_groups": "/rci/object-group/fqdn", "dns_routes": "/rci/dns-proxy/route", "info": "/rci/show/version",
}
var collectorOrder = []string{"system", "dns", "interfaces", "hosts", "static_routes", "dns_groups", "dns_routes", "info"}

// New prepares isolated router sessions without contacting any router.
// Configurations normally come from config.LoadExporterConfig.
func New(cfg *config.ExporterConfig) (*Exporter, error) {
	if cfg == nil || cfg.MaxConcurrentRouters < 1 || cfg.Timeout <= 0 || cfg.MaxBackoff <= 0 || len(cfg.Routers) == 0 {
		return nil, errors.New("invalid exporter settings")
	}
	e := &Exporter{cfg: *cfg, slots: make(chan struct{}, cfg.MaxConcurrentRouters)}
	for _, router := range cfg.Routers {
		session, err := gokeenrestapi.Common.NewMonitoringSession(router.Connection, cfg.Timeout)
		if err != nil {
			for _, t := range e.targets {
				t.session.Close()
			}
			return nil, fmt.Errorf("router %q: %w", router.Name, err)
		}
		t := &target{name: router.Name, interfaces: slices.Clone(router.Interfaces), session: session}
		for _, name := range collectorOrder {
			key := name
			if name == "static_routes" || name == "dns_groups" || name == "dns_routes" {
				key = "routing"
			}
			settings, ok := cfg.Collectors[key]
			if !ok || (settings.Enabled != nil && !*settings.Enabled) {
				continue
			}
			if settings.Interval <= 0 || settings.MaxAge < settings.Interval {
				session.Close()
				for _, target := range e.targets {
					target.session.Close()
				}
				return nil, errors.New("invalid collector interval or max_age")
			}
			t.collectors = append(t.collectors, &collectorState{name: name, settings: settings})
		}
		if len(t.collectors) == 0 {
			session.Close()
			for _, target := range e.targets {
				target.session.Close()
			}
			return nil, errors.New("no enabled collectors")
		}
		e.targets = append(e.targets, t)
	}
	registry := prometheus.NewRegistry()
	if err := registry.Register(e); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{MaxRequestsInFlight: 64, Timeout: 10 * time.Second}))
	e.handler = mux
	return e, nil
}

// Handler returns the HTTP handler; requests never perform router I/O.
func (e *Exporter) Handler() http.Handler { return e.handler }

// Run polls until cancellation and waits for all router workers to stop.
// Call once per exporter; HTTP handlers may be used concurrently.
func (e *Exporter) Run(ctx context.Context) {
	var wg sync.WaitGroup
	spread := 30 * time.Second
	for _, t := range e.targets {
		for _, c := range t.collectors {
			spread = min(spread, c.settings.Interval)
		}
	}
	start := time.Now()
	for i, t := range e.targets {
		wg.Go(func() {
			defer t.session.Close()
			next := start.Add(time.Duration(i) * spread / time.Duration(len(e.targets)))
			for {
				delay := max(time.Until(next), 0)
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				if ctx.Err() != nil {
					return
				}
				e.poll(ctx, i, time.Now())
				t.mu.RLock()
				next = t.collectors[0].next
				for _, c := range t.collectors {
					if c.next.Before(next) {
						next = c.next
					}
				}
				t.mu.RUnlock()
			}
		})
	}
	wg.Wait()
}

// Serve binds the configured listener and runs the HTTP server and pollers.
func (e *Exporter) Serve(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	listener, err := net.Listen("tcp", e.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen for exporter metrics: %w", err)
	}
	return e.serveListener(ctx, listener)
}

func (e *Exporter) serveListener(ctx context.Context, listener net.Listener) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	server := &http.Server{Handler: e.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	pollDone := make(chan struct{})
	go func() { defer close(pollDone); e.Run(runCtx) }()
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serverDone:
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	<-pollDone
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr)
}

func (e *Exporter) poll(ctx context.Context, index int, now time.Time) {
	t := e.targets[index]
	t.pollMu.Lock()
	defer t.pollMu.Unlock()
	var due []*collectorState
	t.mu.RLock()
	for _, c := range t.collectors {
		if !now.Before(c.next) {
			due = append(due, c)
		}
	}
	t.mu.RUnlock()
	if len(due) == 0 {
		return
	}
	select {
	case e.slots <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-e.slots }()
	pollCtx, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
	defer cancel()
	started := time.Now()
	queries := make([]gokeenrestapi.MonitoringQuery, len(due))
	for i, c := range due {
		queries[i].Path = collectorPaths[c.name]
	}
	results := t.session.Read(pollCtx, queries)
	for i, c := range due {
		result := results[i]
		var samples []sample
		err := result.Err
		if err == nil {
			if c.name == "interfaces" {
				samples, err = e.interfaceSamples(pollCtx, t, result.Data)
			} else {
				samples, err = parseSamples(c.name, []json.RawMessage{result.Data}, nil)
			}
		}
		finished := time.Now()
		t.mu.Lock()
		wasSuccess, hadAttempt := c.success, !c.attempted.IsZero()
		c.attempted = finished
		c.duration = finished.Sub(started)
		c.success = err == nil
		interval := c.settings.Interval
		if err == nil {
			c.failures = 0
			c.lastSuccess = finished
			c.samples = samples
		} else {
			c.failures++
			c.errors++
			interval = retryInterval(interval, c.failures, e.cfg.MaxBackoff)
		}
		scheduleBase := finished
		if now.After(finished) {
			scheduleBase = now
		}
		c.next = scheduleBase.Add(interval)
		t.mu.Unlock()
		if err != nil && (wasSuccess || !hadAttempt) {
			gokeenlog.InfoSubStepf("Router %q, collector %s: %v", t.name, c.name, err)
		}
		if err == nil && hadAttempt && !wasSuccess {
			gokeenlog.InfoSubStepf("Router %q, collector %s recovered", t.name, c.name)
		}
	}
}

func retryInterval(interval time.Duration, failures uint, maximum time.Duration) time.Duration {
	capInterval := max(interval, maximum)
	for failures > 0 && interval < capInterval {
		interval = min(interval*2, capInterval)
		failures--
	}
	return interval
}

// Describe leaves this dynamic collector unchecked: series appear only after a
// successful read and disappear when their independently configured max_age expires.
func (*Exporter) Describe(_ chan<- *prometheus.Desc) {}

// Collect copies cached state only. It never contacts routers or refreshes sessions.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	now := time.Now()
	for _, t := range e.targets {
		var metrics []sample
		t.mu.RLock()
		for _, c := range t.collectors {
			labels := map[string]string{"collector": c.name}
			success := 0.0
			if c.success {
				success = 1
			}
			fresh := !c.lastSuccess.IsZero() && now.Sub(c.lastSuccess) <= c.settings.MaxAge
			freshValue := 0.0
			if fresh {
				freshValue = 1
			}
			metrics = append(metrics,
				gauge("keenetic_collector_success", "Whether the last collection attempt succeeded.", success, labels),
				gauge("keenetic_collector_fresh", "Whether a successful snapshot is within max_age.", freshValue, labels),
				gauge("keenetic_collector_duration_seconds", "Duration of the collection cycle through this collector.", c.duration.Seconds(), labels),
				counter("keenetic_collector_errors_total", "Failed collection attempts.", float64(c.errors), labels),
			)
			if !c.lastSuccess.IsZero() {
				metrics = append(metrics, gauge("keenetic_collector_last_success_timestamp_seconds", "Unix time of the last successful collection.", float64(c.lastSuccess.UnixNano())/1e9, labels))
			}
			if !c.attempted.IsZero() {
				metrics = append(metrics, gauge("keenetic_collector_last_attempt_timestamp_seconds", "Unix time of the last collection attempt.", float64(c.attempted.UnixNano())/1e9, labels))
			}
			if fresh {
				metrics = append(metrics, c.samples...)
			}
		}
		t.mu.RUnlock()
		metrics = append(metrics, counter("keenetic_api_requests_total", "Actual HTTP attempts, including authentication and retries.", float64(t.session.Requests()), nil))
		for _, s := range metrics {
			names := []string{"router"}
			for label := range s.labels {
				names = append(names, label)
			}
			slices.Sort(names)
			values := make([]string, len(names))
			for i, name := range names {
				if name == "router" {
					values[i] = t.name
				} else {
					values[i] = s.labels[name]
				}
			}
			desc := prometheus.NewDesc(s.name, s.help, names, nil)
			metric, err := prometheus.NewConstMetric(desc, s.kind, s.value, values...)
			if err != nil {
				ch <- prometheus.NewInvalidMetric(desc, err)
			} else {
				ch <- metric
			}
		}
	}
}
