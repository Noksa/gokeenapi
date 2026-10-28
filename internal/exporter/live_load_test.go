package exporter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type loadReading struct {
	At             time.Time `json:"at"`
	Phase          string    `json:"phase"`
	CPUPercent     float64   `json:"cpu_percent"`
	FreeBytes      uint64    `json:"free_bytes"`
	AvailableBytes uint64    `json:"available_bytes"`
	UsedBytes      uint64    `json:"used_bytes"`
	APIRequests    uint64    `json:"api_requests"`
}

type linuxLoad struct{ total, idle, memory, free, available, buffers, cached uint64 }

func loadHasSnapshot(c *collectorState) bool { return !c.lastSuccess.IsZero() }

func openLoadReport(path string) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("report path must be absolute")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("report destination must be a new writable file in an existing directory")
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("cannot set private report permissions")
	}
	return file, nil
}

func parseLinuxLoad(line string) (linuxLoad, error) {
	var reading linuxLoad
	n, err := fmt.Sscan(line, &reading.total, &reading.idle, &reading.memory, &reading.free, &reading.available, &reading.buffers, &reading.cached)
	if err != nil || n != 7 || len(strings.Fields(line)) != 7 || reading.idle > reading.total || reading.free+reading.buffers+reading.cached > reading.memory {
		return linuxLoad{}, fmt.Errorf("invalid kernel load sample")
	}
	return reading, nil
}

var _ = Describe("Router load sampler", func() {
	DescribeTable("does not overwrite existing files or follow report links", func(kind string) {
		dir := GinkgoT().TempDir()
		target := filepath.Join(dir, "config.yaml")
		Expect(os.WriteFile(target, []byte("private fixture"), 0o644)).To(Succeed())
		path := target
		if kind == "symlink" {
			path = filepath.Join(dir, "report.json")
			Expect(os.Symlink(target, path)).To(Succeed())
		}
		if kind == "hardlink" {
			path = filepath.Join(dir, "report.json")
			Expect(os.Link(target, path)).To(Succeed())
		}
		file, err := openLoadReport(path)
		if file != nil {
			Expect(file.Close()).To(Succeed())
		}
		Expect(err).To(HaveOccurred())
		content, err := os.ReadFile(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("private fixture"))
	}, Entry("existing config or report", "regular"), Entry("symlink to config", "symlink"), Entry("hardlink to config", "hardlink"))
	It("requires an absolute output and creates a new private report", func() {
		path := filepath.Join(GinkgoT().TempDir(), "report.json")
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		relative, err := filepath.Rel(cwd, path)
		Expect(err).NotTo(HaveOccurred())
		file, err := openLoadReport(relative)
		if file != nil {
			Expect(file.Close()).To(Succeed())
			Expect(os.Remove(path)).To(Succeed())
		}
		Expect(err).To(HaveOccurred())
		file, err = openLoadReport(path)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(file.Close)
		info, err := file.Stat()
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o600)))
	})
	It("keeps a successful measurement valid when shutdown cancels an active refresh", func() {
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{"/rci/show/system": {Body: `{"cpuload":10}`, Delay: 20 * time.Millisecond}}))
		defer server.Close()
		defer gokeenrestapi.CleanupTestConfig()
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "dns")
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		now := time.Now()
		e.poll(context.Background(), 0, now)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { defer close(done); e.poll(ctx, 0, now.Add(time.Minute)) }()
		Eventually(func() uint64 { return e.targets[0].session.Requests() }).WithTimeout(time.Second).Should(Equal(uint64(2)))
		cancel()
		<-done
		c := e.targets[0].collectors[0]
		Expect(c.success).To(BeFalse())
		Expect(loadHasSnapshot(c)).To(BeTrue())
	})
	It("parses kernel counters and rejects malformed samples", func() {
		r, err := parseLinuxLoad("1000 200 500000 90000 230000 6000 167000")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.total).To(Equal(uint64(1000)))
		Expect(r.memory - r.free - r.buffers - r.cached).To(Equal(uint64(237000)))
		_, err = parseLinuxLoad("broken")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Live exporter router load", Serial, Label("live"), func() {
	It("compares baseline, all collectors, cached scrapes and recovery with the Compose exporter suspended", func() {
		path := os.Getenv("GOKEENAPI_LIVE_EXPORTER_CONFIG")
		if path == "" {
			Skip("opt-in: set GOKEENAPI_LIVE_EXPORTER_CONFIG, GOKEENAPI_LIVE_ROUTER_SSH and GOKEENAPI_LIVE_REPORT")
		}
		sshTarget, reportPath := os.Getenv("GOKEENAPI_LIVE_ROUTER_SSH"), os.Getenv("GOKEENAPI_LIVE_REPORT")
		Expect(sshTarget).NotTo(BeEmpty())
		Expect(strings.HasPrefix(sshTarget, "-")).To(BeFalse())
		Expect(reportPath).NotTo(BeEmpty())
		cfg, err := config.LoadExporterConfig(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Routers).To(HaveLen(1), "measure one router at a time")
		for name, c := range cfg.Collectors {
			Expect(c.Enabled == nil || *c.Enabled).To(BeTrue(), "enable collector "+name)
		}
		report, err := openLoadReport(reportPath)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(report.Close)
		project := os.Getenv("EXPORTER_PROJECT")
		if project == "" {
			project = "gokeenapi-exporter-local"
		}
		command := func(args ...string) string {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Docker control failed")
			return strings.TrimSpace(string(out))
		}
		ids := strings.Fields(command("ps", "--filter", "label=com.docker.compose.service=exporter", "--format", "{{.ID}}"))
		Expect(ids).To(HaveLen(1), "exactly one running Compose exporter must exist before the experiment")
		container := ids[0]
		Expect(command("inspect", "--format", `{{index .Config.Labels "com.docker.compose.project"}}`, container)).To(Equal(project))
		// Register restoration before stopping: assertions and interruption also restore it.
		DeferCleanup(func() { command("start", container) })
		command("stop", "--time", "15", container)
		Expect(command("inspect", "--format", "{{.State.Running}}", container)).To(Equal("false"))
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		var phase atomic.Value
		phase.Store("baseline")
		var mu sync.Mutex
		var readings []loadReading
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		// One persistent SSH session, no RCI calls. Exclude guest CPU double counting.
		remote := `set -e; while :; do awk '/^cpu / {t=0; for(i=2;i<=9;i++)t+=$i; idle=$5+$6} /^MemTotal:/ {m=$2} /^MemFree:/ {f=$2} /^MemAvailable:/ {a=$2} /^Buffers:/ {b=$2} /^Cached:/ {c=$2} END {printf "%.0f %.0f %.0f %.0f %.0f %.0f %.0f\n",t,idle,m,f,a,b,c}' /proc/stat /proc/meminfo; usleep 250000; done`
		ssh := exec.CommandContext(ctx, "ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=8", sshTarget, remote)
		pipe, err := ssh.StdoutPipe()
		Expect(err).NotTo(HaveOccurred())
		Expect(ssh.Start()).To(Succeed())
		done := make(chan error, 1)
		go func() {
			scanner := bufio.NewScanner(pipe)
			var previous linuxLoad
			previousPhase := ""
			for scanner.Scan() {
				current, err := parseLinuxLoad(scanner.Text())
				if err != nil {
					done <- err
					return
				}
				p := phase.Load().(string)
				if previous.total > 0 && current.total > previous.total && current.idle >= previous.idle && previousPhase == p {
					total, idle := current.total-previous.total, current.idle-previous.idle
					if idle <= total && current.free+current.buffers+current.cached <= current.memory {
						row := loadReading{time.Now(), p, 100 * (1 - float64(idle)/float64(total)), current.free * 1024, current.available * 1024, (current.memory - current.free - current.buffers - current.cached) * 1024, e.targets[0].session.Requests()}
						mu.Lock()
						readings = append(readings, row)
						mu.Unlock()
					}
				}
				previous, previousPhase = current, p
			}
			done <- scanner.Err()
		}()
		DeferCleanup(func() { cancel(); _ = ssh.Wait(); _ = <-done })
		waitPhase := func(name string, duration time.Duration) {
			By(name)
			phase.Store(name)
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-timer.C:
			case err := <-done:
				done <- err
				Fail(fmt.Sprintf("SSH sampler stopped: %v", err))
			}
		}
		waitPhase("baseline", 30*time.Second)
		runCtx, stop := context.WithCancel(context.Background())
		runDone := make(chan struct{})
		phase.Store("polling")
		go func() { defer close(runDone); e.Run(runCtx) }()
		DeferCleanup(func() { stop(); <-runDone })
		waitPhase("polling", 120*time.Second)
		stop()
		<-runDone
		for _, c := range e.targets[0].collectors {
			Expect(loadHasSnapshot(c)).To(BeTrue(), "collector "+c.name+" has no successful snapshot")
		}
		requests := e.targets[0].session.Requests()
		var scrapes atomic.Uint64
		scrapeDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(scrapeDone)
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				_ = scrape(e)
				scrapes.Add(1)
				time.Sleep(10 * time.Millisecond)
			}
		}()
		waitPhase("cached_scrapes", 30*time.Second)
		<-scrapeDone
		Expect(e.targets[0].session.Requests()).To(Equal(requests), "cached scrapes must cause zero router requests")
		waitPhase("recovery", 30*time.Second)
		mu.Lock()
		rows := slices.Clone(readings)
		mu.Unlock()
		for _, name := range []string{"baseline", "polling", "cached_scrapes", "recovery"} {
			var cpu []float64
			minAvailable, maxUsed := uint64(^uint64(0)), uint64(0)
			for _, r := range rows {
				if r.Phase == name {
					cpu = append(cpu, r.CPUPercent)
					minAvailable = min(minAvailable, r.AvailableBytes)
					maxUsed = max(maxUsed, r.UsedBytes)
				}
			}
			Expect(len(cpu)).To(BeNumerically(">", 50), "insufficient samples: "+name)
			slices.Sort(cpu)
			sum := 0.0
			for _, v := range cpu {
				sum += v
			}
			GinkgoWriter.Printf("%s: n=%d CPU mean=%.2f%% p95=%.2f%% max=%.2f%%; min available=%.2f MiB max used=%.2f MiB\n", name, len(cpu), sum/float64(len(cpu)), cpu[int(float64(len(cpu)-1)*0.95)], cpu[len(cpu)-1], float64(minAvailable)/(1<<20), float64(maxUsed)/(1<<20))
		}
		data, err := json.MarshalIndent(map[string]any{"samples": rows, "api_requests": requests, "cached_scrapes": scrapes.Load(), "suspended_container": container, "poll_intervals": cfg.Collectors}, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		_, err = report.Write(data)
		Expect(err).NotTo(HaveOccurred())
		AddReportEntry("router-load-report", filepath.Base(reportPath))
	})
})
