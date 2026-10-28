package exporter

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func dnsMetadataFixture(configuration string, tls, https any) json.RawMessage {
	body, err := json.Marshal(map[string]any{"proxy-status": []any{map[string]any{
		"proxy-name": "Policy0", "proxy-config": configuration, "proxy-tls": tls, "proxy-https": https,
		"proxy-stat": "Total incoming requests: 10\n127.0.0.1 40516 8 7 0 10ms 10ms 1\n192.0.2.53 53 2 2 0 1ms 1ms 1\n",
	}}})
	Expect(err).NotTo(HaveOccurred())
	return body
}

var _ = Describe("Automatic router metadata", func() {
	AfterEach(gokeenrestapi.CleanupTestConfig)
	It("keeps gateway-only routes visible with an explicit unspecified-interface label", func() {
		samples, err := parseSamples("static_routes", []json.RawMessage{json.RawMessage(`[{"network":"192.0.2.0","mask":"255.255.255.0","gateway":"192.168.1.1"}]`)}, nil)
		Expect(err).NotTo(HaveOccurred())
		t := &target{}
		_, ok := sampleValue(samples, "keenetic_static_routes", nil)
		Expect(ok).To(BeTrue())
		for _, s := range samples {
			if s.name == "keenetic_static_routes" {
				named := t.namedSample(s, time.Now())
				Expect(named.labels).To(Equal(map[string]string{"interface": "", "interface_name": "Unspecified interface"}))
				Expect(named.value).To(Equal(float64(1)))
				Expect(s.labels).To(Equal(map[string]string{"interface": ""}))
			}
		}
	})
	It("reports used memory with the same cache and buffer exclusion as Keenetic", func() {
		samples, err := parseSamples("system", []json.RawMessage{json.RawMessage(`{"memtotal":524288,"memfree":93552,"memcache":167072,"membuffers":6352}`)}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, ok := sampleValue(samples, "keenetic_memory_used_bytes", nil)
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(257312 * 1024)))
	})
	It("does not invent used memory when a component is missing or inconsistent", func() {
		for _, body := range []string{`{"memtotal":100,"memfree":10}`, `{"memtotal":100,"memfree":80,"memcache":30,"membuffers":0}`} {
			samples, err := parseSamples("system", []json.RawMessage{json.RawMessage(body)}, nil)
			Expect(err).NotTo(HaveOccurred())
			_, ok := sampleValue(samples, "keenetic_memory_used_bytes", nil)
			Expect(ok).To(BeFalse())
		}
	})
	It("names TLS upstreams and combines configured zones without multiplying counters", func() {
		body := dnsMetadataFixture("dns_server = 127.0.0.1:40516 . # 1.1.1.1\ndns_server = 192.0.2.53 lab.example\ndns_server = 192.0.2.53 corp.example\ndns_server = 192.0.2.53 corp.example\n",
			map[string]any{"server-tls": []any{map[string]any{"address": "1.1.1.1", "sni": "cloudflare-dns.com"}}}, nil)
		samples, err := parseSamples("dns", []json.RawMessage{body}, nil)
		Expect(err).NotTo(HaveOccurred())
		count := 0
		for _, s := range samples {
			if s.name != "keenetic_dns_upstream_requests_total" {
				continue
			}
			count++
			if s.labels["port"] == "40516" {
				Expect(s.labels["upstream_name"]).To(Equal("DoT cloudflare-dns.com (1.1.1.1:853) / default"))
			} else {
				Expect(s.labels["upstream_name"]).To(Equal("DNS 192.0.2.53:53 / corp.example, lab.example"))
			}
			Expect(s.labels).To(HaveLen(4))
		}
		Expect(count).To(Equal(2))
		Expect(samples).To(HaveLen(9)) // existing counters and durations only
	})
	It("keeps DNS statistics when optional metadata is malformed or ambiguous", func() {
		body := dnsMetadataFixture("dns_server = 127.0.0.1:40516 . # 1.1.1.1\n", "unexpected firmware format", nil)
		samples, err := parseSamples("dns", []json.RawMessage{body}, nil)
		Expect(err).NotTo(HaveOccurred())
		_, ok := sampleValue(samples, "keenetic_dns_upstream_requests_total", map[string]string{"upstream_name": "DNS 127.0.0.1:40516"})
		Expect(ok).To(BeTrue())
		body = dnsMetadataFixture("dns_server = 127.0.0.1:40516 . # 1.1.1.1\n", map[string]any{"server-tls": []any{
			map[string]any{"address": "1.1.1.1", "sni": "one.example"}, map[string]any{"address": "1.1.1.1", "sni": "two.example"},
		}}, nil)
		samples, err = parseSamples("dns", []json.RawMessage{body}, nil)
		Expect(err).NotTo(HaveOccurred())
		_, ok = sampleValue(samples, "keenetic_dns_upstream_requests_total", map[string]string{"upstream_name": "DNS 127.0.0.1:40516"})
		Expect(ok).To(BeTrue())
	})
	It("never publishes credentials or query tokens from DoH metadata", func() {
		body := dnsMetadataFixture("dns_server = 127.0.0.1:40516 . # https://user:password@dns.example/dns-query?token=private\n", nil,
			map[string]any{"server-https": []any{map[string]any{"url": "https://user:password@dns.example/dns-query?token=private"}}})
		samples, err := parseSamples("dns", []json.RawMessage{body}, nil)
		Expect(err).NotTo(HaveOccurred())
		var names []string
		for _, s := range samples {
			names = append(names, s.labels["upstream_name"])
		}
		Expect(strings.Join(names, " ")).NotTo(Or(ContainSubstring("password"), ContainSubstring("private"), ContainSubstring("user:")))
		Expect(names).To(ContainElement("DoH dns.example / default"))
	})
	It("preserves IPv6 DoH brackets and ports", func() {
		origin := "https://[2001:db8::1]:8443/dns-query"
		body := dnsMetadataFixture("dns_server = 127.0.0.1:40516 . # "+origin+"\n", nil, map[string]any{"server-https": []any{map[string]any{"url": origin}}})
		samples, err := parseSamples("dns", []json.RawMessage{body}, nil)
		Expect(err).NotTo(HaveOccurred())
		_, ok := sampleValue(samples, "keenetic_dns_upstream_requests_total", map[string]string{"upstream_name": "DoH [2001:db8::1]:8443 / default"})
		Expect(ok).To(BeTrue())
	})
	It("ignores unsupported optional interface-name formats while retaining counters", func() {
		for _, name := range []string{`1`, `{}`} {
			samples, err := parseSamples("interfaces", []json.RawMessage{json.RawMessage(`{"Wireguard0":{"id":"Wireguard0","interface-name":` + name + `,"type":"Wireguard"}}`), json.RawMessage(`{"rxbytes":42}`)}, []string{"Wireguard0"})
			Expect(err).NotTo(HaveOccurred())
			v, ok := sampleValue(samples, "keenetic_interface_receive_bytes_total", nil)
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(float64(42)))
		}
	})
	It("refreshes policies independently of a long DNS interval and never retries metadata faster than five minutes", func() {
		for _, status := range []int{200, 404} {
			var policies, dns atomic.Int64
			server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
				"/rci/show/dns-proxy": {Body: `{"proxy-status":[{"proxy-name":"Policy0","proxy-stat":"Total incoming requests: 10\n"}]}`},
				"/rci/ip/policy":      {Body: `{"Policy0":{"description":"VPN"}}`, StatusCode: status},
			}), gokeenrestapi.WithMonitoringRequestObserver(func(_, path string) {
				if path == "/rci/ip/policy" {
					policies.Add(1)
				}
				if path == "/rci/" {
					dns.Add(1)
				}
			}))
			cfg := fixtureConfig(server.URL)
			delete(cfg.Collectors, "system")
			cfg.Collectors["dns"] = config.ExporterCollector{Interval: 15 * time.Minute, MaxAge: 30 * time.Minute}
			e, err := New(cfg)
			Expect(err).NotTo(HaveOccurred())
			now := time.Now()
			e.poll(context.Background(), 0, now)
			e.poll(context.Background(), 0, now.Add(2*time.Minute))
			Expect(policies.Load()).To(Equal(int64(1)))
			e.poll(context.Background(), 0, now.Add(6*time.Minute))
			Expect(policies.Load()).To(Equal(int64(2)))
			Expect(dns.Load()).To(Equal(int64(1)))
			server.Close()
		}
	})
	It("uses descriptions throughout interface and route metrics and exports physical speed without extra counter reads", func() {
		var requests atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/interface":                      {Body: `{"Wireguard0":{"id":"Wireguard0","type":"Wireguard","description":"VPN","interface-name":"alias","state":"up","connected":"yes"},"1":{"id":"GigabitEthernet0/0","interface-name":"1","type":"Port","speed":"1000","link":"up"}}`},
			"/rci/show/interface/stat?name=Wireguard0": {Body: `{"rxbytes":1024}`},
			"/rci/ip/route":                            {Body: `[{"host":"192.0.2.1","interface":"Wireguard0"}]`},
			"/rci/object-group/fqdn":                   {Body: `{}`}, "/rci/dns-proxy/route": {Body: `[]`},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(string, string) { requests.Add(1) }))
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		cfg.Collectors = map[string]config.ExporterCollector{"interfaces": {Interval: 30 * time.Second, MaxAge: time.Minute}, "routing": {Interval: 5 * time.Minute, MaxAge: 10 * time.Minute}}
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		e.poll(context.Background(), 0, time.Now())
		body := scrape(e)
		Expect(body).To(ContainSubstring(`keenetic_interface_receive_bytes_total{interface="Wireguard0",interface_name="VPN (Wireguard0)",router="home"} 1024`))
		Expect(body).To(ContainSubstring(`keenetic_static_routes{interface="Wireguard0",interface_name="VPN (Wireguard0)",router="home"} 1`))
		Expect(body).To(ContainSubstring(`keenetic_interface_speed_bits_per_second{interface="GigabitEthernet0/0",interface_name="1 (GigabitEthernet0/0)",router="home"} 1e+09`))
		Expect(body).NotTo(ContainSubstring("keenetic_interface_info"))
		Expect(requests.Load()).To(Equal(int64(5)))
		for range 10 {
			_ = scrape(e)
		}
		Expect(requests.Load()).To(Equal(int64(5)))
	})
	It("reads policy descriptions infrequently and keeps one series per proxy", func() {
		var policies atomic.Int64
		server := gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithMonitoringResponses(map[string]gokeenrestapi.MockMonitoringResponse{
			"/rci/show/dns-proxy": {Body: `{"proxy-status":[{"proxy-name":"Policy0","proxy-stat":"Total incoming requests: 10\n"}]}`},
			"/rci/ip/policy":      {Body: `{"Policy0":{"description":"Only VPN"}}`},
		}), gokeenrestapi.WithMonitoringRequestObserver(func(_, path string) {
			if path == "/rci/ip/policy" {
				policies.Add(1)
			}
		}))
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		delete(cfg.Collectors, "system")
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		now := time.Now()
		e.poll(context.Background(), 0, now)
		Expect(scrape(e)).To(ContainSubstring(`keenetic_dns_requests_total{proxy="Policy0",proxy_name="Only VPN (Policy0)",router="home"} 10`))
		e.poll(context.Background(), 0, now.Add(time.Minute))
		Expect(policies.Load()).To(Equal(int64(1)))
		e.poll(context.Background(), 0, now.Add(6*time.Minute))
		Expect(policies.Load()).To(Equal(int64(2)))
	})
	It("falls back after metadata expiry, keeps counter values and does not mutate cached samples", func() {
		server := gokeenrestapi.SetupMockRouterForTest()
		defer server.Close()
		cfg := fixtureConfig(server.URL)
		e, err := New(cfg)
		Expect(err).NotTo(HaveOccurred())
		t := e.targets[0]
		now := time.Now()
		t.mu.Lock()
		t.interfaceNames = map[string]string{"Wireguard0": "VPN (Wireguard0)"}
		t.interfaceNamesAt = now
		t.interfaceNamesMaxAge = time.Minute
		t.policyNames = map[string]string{"Policy0": "Only VPN (Policy0)"}
		t.policyNamesAt = now
		t.collectors[0].lastSuccess = now
		t.collectors[0].samples = []sample{counter("keenetic_interface_receive_bytes_total", "Test counter.", 42, map[string]string{"interface": "Wireguard0"})}
		t.collectors[1].lastSuccess = now
		t.collectors[1].samples = []sample{counter("keenetic_dns_requests_total", "Test DNS counter.", 10, map[string]string{"proxy": "Policy0"})}
		t.mu.Unlock()
		body := scrape(e)
		Expect(body).To(ContainSubstring(`interface_name="VPN (Wireguard0)"`))
		Expect(body).To(ContainSubstring(`proxy_name="Only VPN (Policy0)"`))
		t.mu.Lock()
		t.interfaceNamesAt = now.Add(-2 * time.Minute)
		t.policyNamesAt = now.Add(-11 * time.Minute)
		t.mu.Unlock()
		body = scrape(e)
		Expect(body).To(ContainSubstring(`keenetic_interface_receive_bytes_total{interface="Wireguard0",interface_name="Wireguard0",router="home"} 42`))
		Expect(body).To(ContainSubstring(`keenetic_dns_requests_total{proxy="Policy0",proxy_name="Policy0",router="home"} 10`))
		Expect(t.collectors[0].samples[0].labels).To(Equal(map[string]string{"interface": "Wireguard0"}))
	})
	DescribeTable("chooses display names without losing IDs", func(id, description, name, want string) {
		Expect(displayName(id, description, name)).To(Equal(want))
	}, Entry("description", "Wireguard0", "VPN", "alias", "VPN (Wireguard0)"), Entry("name fallback", "Bridge0", " ", "Home", "Home (Bridge0)"), Entry("ID fallback", "Bridge0", "", "", "Bridge0"))
})
