package exporter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func fixtureConfig(url string) *config.ExporterConfig {
	return &config.ExporterConfig{Listen: "127.0.0.1:9479", Timeout: time.Second, MaxConcurrentRouters: 2, MaxBackoff: 4 * time.Minute,
		Collectors: map[string]config.ExporterCollector{"system": {Interval: 30 * time.Second, MaxAge: time.Minute}, "dns": {Interval: 30 * time.Second, MaxAge: time.Minute}},
		Routers:    []config.ExporterRouter{{Name: "home", Connection: config.Keenetic{URL: url, Login: "monitor", Password: "secret"}}},
	}
}

func scrape(e *Exporter) string {
	r := httptest.NewRecorder()
	e.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	Expect(r.Code).To(Equal(http.StatusOK))
	return r.Body.String()
}

var _ = Describe("Exporter polling", func() {
	AfterEach(gokeenrestapi.CleanupTestConfig)
	It("batches due collectors and serves concurrent scrapes without additional router requests", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system":    {Body: `{"cpuload":25,"uptime":"42"}`},
			"/rci/show/dns-proxy": {Body: `{"proxy-status":[{"proxy-name":"System","proxy-stat":"Total incoming requests: 100\n"}]}`},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }))
		defer server.Close()
		e, err := New(fixtureConfig(server.URL))
		Expect(err).NotTo(HaveOccurred())
		e.poll(context.Background(), 0, time.Now())
		Expect(requests.Load()).To(Equal(int64(1)))
		Expect(scrape(e)).To(ContainSubstring(`keenetic_dns_requests_total{proxy="System",router="home"} 100`))
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() { defer GinkgoRecover(); Expect(scrape(e)).To(ContainSubstring("keenetic_cpu_load_ratio")) })
		}
		wg.Wait()
		Expect(requests.Load()).To(Equal(int64(1)))
	})
	It("prevents overlapping/queued polls of the same due data", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system": {Body: `{"cpuload":1}`, Delay: 20 * time.Millisecond},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }))
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "dns")
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		var wg sync.WaitGroup
		now := time.Now()
		for range 10 {
			wg.Go(func() { e.poll(context.Background(), 0, now) })
		}
		wg.Wait()
		Expect(requests.Load()).To(Equal(int64(1)))
	})
	It("reports partial failure without masking successful collectors or inventing DNS zeros", func() {
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system":    {Body: `{"cpuload":10}`},
			"/rci/show/dns-proxy": {Body: `{"status":[{"status":"error","code":"unsupported"}]}`},
		}))
		defer server.Close()
		e, err := New(fixtureConfig(server.URL))
		Expect(err).NotTo(HaveOccurred())
		e.poll(context.Background(), 0, time.Now())
		body := scrape(e)
		Expect(body).To(ContainSubstring("keenetic_cpu_load_ratio"))
		Expect(body).To(ContainSubstring(`keenetic_collector_success{collector="dns",router="home"} 0`))
		Expect(body).NotTo(ContainSubstring("keenetic_dns_requests_total"))
	})
	It("expires successful snapshots after failures and increases retry intervals", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system": {Body: `{"cpuload":10}`},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }))
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "dns")
		cfg.Collectors["system"] = config.ExporterCollector{Interval: 5 * time.Millisecond, MaxAge: 20 * time.Millisecond}
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		defer server.Close()
		e.poll(context.Background(), 0, time.Now())
		Expect(scrape(e)).To(ContainSubstring("keenetic_cpu_load_ratio"))
		server.Close()
		time.Sleep(6 * time.Millisecond)
		e.poll(context.Background(), 0, time.Now())
		time.Sleep(20 * time.Millisecond)
		body := scrape(e)
		Expect(body).NotTo(ContainSubstring("keenetic_cpu_load_ratio"))
		Expect(body).To(ContainSubstring(`keenetic_collector_success{collector="system",router="home"} 0`))
	})
	It("schedules slow collectors separately", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system": {Body: `{"cpuload":1}`},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }))
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "dns")
		cfg.Collectors["routing"] = config.ExporterCollector{Interval: 5 * time.Minute, MaxAge: 10 * time.Minute}
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		now := time.Now()
		e.poll(context.Background(), 0, now)
		Expect(requests.Load()).To(Equal(int64(4)))
		e.poll(context.Background(), 0, now.Add(31*time.Second))
		Expect(requests.Load()).To(Equal(int64(5)))
	})
	It("starts while a router is unavailable and drains workers on cancellation", func() {
		e, err := New(fixtureConfig("http://127.0.0.1:1"))
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { defer close(done); e.Run(ctx) }()
		Expect(scrape(e)).To(ContainSubstring("keenetic_collector_success"))
		cancel()
		Eventually(done).WithTimeout(time.Second).Should(BeClosed())
	})
})

var _ = Describe("Exporter concurrency and scheduling", func() {
	AfterEach(gokeenrestapi.CleanupTestConfig)
	It("bounds active router polls and keeps their samples separate", func() {
		var active, maximum atomic.Int64
		var servers []*httptest.Server
		cfg := fixtureConfig("http://127.0.0.1:1")
		delete(cfg.Collectors, "dns")
		cfg.MaxConcurrentRouters = 2
		cfg.Routers = nil
		for i := range 5 {
			server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
				"/rci/show/system": {Body: fmt.Sprintf(`{"cpuload":%d}`, i+1), Delay: 20 * time.Millisecond},
			}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) {
				n := active.Add(1)
				for old := maximum.Load(); n > old; old = maximum.Load() {
					if maximum.CompareAndSwap(old, n) {
						break
					}
				}
				time.AfterFunc(15*time.Millisecond, func() { active.Add(-1) })
			}))
			servers = append(servers, server)
			cfg.Routers = append(cfg.Routers, config.ExporterRouter{Name: fmt.Sprintf("router%d", i), Connection: config.Keenetic{URL: server.URL, Login: "monitor", Password: "secret"}})
		}
		defer func() {
			for _, server := range servers {
				server.Close()
			}
		}()
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		var wg sync.WaitGroup
		for i := range 5 {
			wg.Go(func() { e.poll(context.Background(), i, time.Now()) })
		}
		wg.Wait()
		Expect(maximum.Load()).To(BeNumerically("<=", 2))
		body := scrape(e)
		for i := range 5 {
			Expect(body).To(ContainSubstring(fmt.Sprintf(`keenetic_cpu_load_ratio{router="router%d"} %g`, i, float64(i+1)/100)))
		}
	})
	DescribeTable("backs off failures within the cap without shortening slow intervals", func(interval time.Duration, failures uint, cap, expected time.Duration) {
		Expect(retryInterval(interval, failures, cap)).To(Equal(expected))
	}, Entry("first failure", 30*time.Second, uint(1), 5*time.Minute, time.Minute), Entry("capped", 30*time.Second, uint(10), 5*time.Minute, 5*time.Minute), Entry("slow collector", time.Hour, uint(3), 5*time.Minute, time.Hour))
	It("cancels an in-flight router read and waits for the worker to stop", func() {
		started := make(chan struct{}, 1)
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/system": {Body: `{"cpuload":1}`, Delay: time.Second},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) {
			select {
			case started <- struct{}{}:
			default:
			}
		}))
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "dns")
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() { defer close(done); e.Run(ctx) }()
		Eventually(started).WithTimeout(time.Second).Should(Receive())
		cancel()
		Eventually(done).WithTimeout(500 * time.Millisecond).Should(BeClosed())
	})
})

var _ = Describe("Exporter interface collection", func() {
	AfterEach(gokeenrestapi.CleanupTestConfig)
	It("discovers layer-three interfaces, batches their counters, and honors an explicit selection", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(
			gokeenrestapi.WithInterfaces([]gokeenrestapi.MockInterface{
				{ID: "ISP", Type: "GigabitEthernet", Address: "192.0.2.1", State: "up", Connected: "yes", DefaultGw: true},
				{ID: "Wireguard0", Type: "Wireguard", State: "down", Connected: "no"},
				{ID: "WifiMaster0", Type: "WifiMaster"},
			}),
			gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
				"/rci/show/interface/stat?name=ISP":        {Body: `{"rxbytes":"1024","txbytes":2048}`},
				"/rci/show/interface/stat?name=Wireguard0": {Body: `{"rxbytes":0,"txbytes":0}`},
			}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }),
		)
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		cfg.Collectors = map[string]config.ExporterCollector{"interfaces": {Interval: 30 * time.Second, MaxAge: time.Minute}}
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		e.poll(context.Background(), 0, time.Now())
		body := scrape(e)
		Expect(requests.Load()).To(Equal(int64(2)))
		Expect(body).To(ContainSubstring(`keenetic_interface_receive_bytes_total{interface="ISP",router="home"} 1024`))
		Expect(body).To(ContainSubstring(`keenetic_interface_connected{interface="Wireguard0",router="home"} 0`))
		Expect(body).NotTo(ContainSubstring(`interface="WifiMaster0"`))
		cfg.Routers[0].Interfaces = []string{"Wireguard0"}
		selected, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		selected.poll(context.Background(), 0, time.Now())
		Expect(scrape(selected)).NotTo(ContainSubstring(`interface="ISP"`))
	})
})
