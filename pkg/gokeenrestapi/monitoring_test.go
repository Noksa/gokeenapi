package gokeenrestapi

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Monitoring sessions", func() {
	AfterEach(CleanupTestConfig)
	connection := func(url string) config.Keenetic {
		return config.Keenetic{URL: url, Login: "monitor", Password: "secret"}
	}
	It("reads configuration through GET without executing a config command", func() {
		server := SetupMockRouterForTest(WithDNSRecords(map[string]string{"example.test": "192.0.2.1"}))
		defer server.Close()
		session, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer session.Close()
		results := session.Read(context.Background(), []MonitoringQuery{{Path: "/rci/ip/route"}, {Path: "/rci/object-group/fqdn"}})
		Expect(results).To(HaveLen(2))
		Expect(results[0].Err).NotTo(HaveOccurred())
		Expect(results[1].Err).NotTo(HaveOccurred())
		Expect(session.Requests()).To(Equal(uint64(2)))
	})
	It("rejects mutations before making any network requests", func() {
		server := SetupMockRouterForTest()
		defer server.Close()
		session, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer session.Close()
		results := session.Read(context.Background(), []MonitoringQuery{{Path: "/rci/system/reboot"}})
		Expect(results[0].Err).To(HaveOccurred())
		Expect(session.Requests()).To(BeZero())
	})
	It("keeps concurrent router results independent of global CLI configuration", func() {
		a := SetupMockRouterForTest(WithVersion("5.1.8"))
		defer a.Close()
		b := SetupMockRouterForTest(WithVersion("4.3.6"))
		defer b.Close()
		sa, err := Common.NewMonitoringSession(connection(a.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		sb, err := Common.NewMonitoringSession(connection(b.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer sa.Close()
		defer sb.Close()
		var wg sync.WaitGroup
		for _, pair := range []struct {
			s       *MonitoringSession
			version string
		}{{sa, "5.1.8"}, {sb, "4.3.6"}} {
			wg.Go(func() {
				defer GinkgoRecover()
				for range 5 {
					r := pair.s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/version"}})
					Expect(r[0].Err).NotTo(HaveOccurred())
					Expect(string(r[0].Data)).To(ContainSubstring(pair.version))
				}
			})
		}
		wg.Wait()
	})
	It("honors cancellation without retrying or exposing credentials", func() {
		session, err := Common.NewMonitoringSession(connection("http://127.0.0.1:1"), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer session.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := session.Read(ctx, []MonitoringQuery{{Path: "/rci/show/version"}})
		Expect(result[0].Err).To(HaveOccurred())
		Expect(result[0].Err.Error()).NotTo(ContainSubstring("secret"))
		Expect(session.Requests()).To(BeZero())
	})
	It("batches show reads into one request and keeps successful resources when another returns an RCI error", func() {
		server := SetupMockRouterForTest(WithMonitoringResponses(map[string]MockMonitoringResponse{
			"/rci/show/system":    {Body: `{"cpuload":3}`},
			"/rci/show/dns-proxy": {Body: `{"status":[{"status":"error","code":"unsupported"}]}`},
		}))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/system"}, {Path: "/rci/show/dns-proxy"}})
		Expect(r[0].Err).NotTo(HaveOccurred())
		Expect(r[1].Err).To(HaveOccurred())
		Expect(s.Requests()).To(Equal(uint64(1)))
	})
	It("falls back once when batching is unsupported and then reuses GET", func() {
		server := SetupMockRouterForTest(WithoutMonitoringBatch())
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		for range 2 {
			r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/version"}})
			Expect(r[0].Err).NotTo(HaveOccurred())
		}
		Expect(s.Requests()).To(Equal(uint64(3)))
	})
	It("authenticates once per router and reuses isolated cookies", func() {
		var wg sync.WaitGroup
		for _, name := range []string{"first", "second"} {
			server := SetupMockRouterForTest(WithMonitoringAuth(name, name+"-password", "session="+name))
			DeferCleanup(server.Close)
			s, err := Common.NewMonitoringSession(config.Keenetic{URL: server.URL, Login: name, Password: name + "-password"}, time.Second)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(s.Close)
			wg.Go(func() {
				defer GinkgoRecover()
				for range 3 {
					r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/version"}})
					Expect(r[0].Err).NotTo(HaveOccurred())
				}
				Expect(s.Requests()).To(Equal(uint64(6)))
			})
		}
		wg.Wait()
	})
	It("rejects malformed successful responses and failed authentication", func() {
		server := SetupMockRouterForTest(WithMonitoringAuth("other-user", "other-password", "session=other"))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/version"}})
		Expect(r[0].Err).To(MatchError(ContainSubstring("authentication failed")))
		Expect(s.Requests()).To(Equal(uint64(3)))
	})
	It("bounds the number of read commands in each HTTP batch", func() {
		server := SetupMockRouterForTest()
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		queries := make([]MonitoringQuery, 205)
		for i := range queries {
			queries[i].Path = "/rci/show/version"
		}
		for _, result := range s.Read(context.Background(), queries) {
			Expect(result.Err).NotTo(HaveOccurred())
		}
		Expect(s.Requests()).To(Equal(uint64(3)))
	})
	It("keeps independent configuration reads after one endpoint returns HTTP 500", func() {
		server := SetupMockRouterForTest(WithMonitoringResponses(map[string]MockMonitoringResponse{
			"/rci/ip/route": {StatusCode: 500, Body: `{}`},
		}))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/ip/route"}, {Path: "/rci/object-group/fqdn"}, {Path: "/rci/dns-proxy/route"}})
		Expect(r[0].Err).To(HaveOccurred())
		Expect(r[1].Err).NotTo(HaveOccurred())
		Expect(r[2].Err).NotTo(HaveOccurred())
		Expect(s.Requests()).To(Equal(uint64(3)))
	})
	It("rejects oversized responses instead of retaining unbounded router data", func() {
		server := SetupMockRouterForTest(WithMonitoringResponses(map[string]MockMonitoringResponse{
			"/rci/ip/route": {Body: `{"padding":"` + strings.Repeat("a", 9<<20) + `"}`},
		}))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/ip/route"}})
		Expect(r[0].Err).To(HaveOccurred())
		Expect(r[0].Data).To(BeEmpty())
	})
	It("keeps independent GETs when a show batch has malformed JSON structure", func() {
		server := SetupMockRouterForTest(WithRciBody([]byte(`{}`)))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/show/system"}, {Path: "/rci/ip/route"}})
		Expect(r[0].Err).To(HaveOccurred())
		Expect(r[1].Err).NotTo(HaveOccurred())
		Expect(s.Requests()).To(Equal(uint64(2)))
	})
	It("keeps independent GETs after a resource-specific response size error", func() {
		server := SetupMockRouterForTest(WithMonitoringResponses(map[string]MockMonitoringResponse{
			"/rci/ip/route": {Body: `{"padding":"` + strings.Repeat("a", 9<<20) + `"}`},
		}))
		defer server.Close()
		s, err := Common.NewMonitoringSession(connection(server.URL), time.Second)
		Expect(err).NotTo(HaveOccurred())
		defer s.Close()
		r := s.Read(context.Background(), []MonitoringQuery{{Path: "/rci/ip/route"}, {Path: "/rci/object-group/fqdn"}})
		Expect(r[0].Err).To(HaveOccurred())
		Expect(r[1].Err).NotTo(HaveOccurred())
	})
})
