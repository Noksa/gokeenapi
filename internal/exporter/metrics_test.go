package exporter

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

func sampleValue(samples []sample, name string, labels map[string]string) (float64, bool) {
	for _, s := range samples {
		if s.name != name {
			continue
		}
		match := true
		for k, v := range labels {
			if s.labels[k] != v {
				match = false
			}
		}
		if match {
			return s.value, true
		}
	}
	return 0, false
}

var _ = Describe("Exporter metrics", func() {
	It("converts system KiB and percent to base units and handles numeric strings", func() {
		samples, err := parseSamples("system", []json.RawMessage{json.RawMessage(`{"cpuload":25,"memtotal":262144,"memfree":131072,"uptime":"12345"}`)}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, ok := sampleValue(samples, "keenetic_memory_total_bytes", nil)
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(268435456)))
		v, ok = sampleValue(samples, "keenetic_cpu_load_ratio", nil)
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(0.25))
		_, ok = sampleValue(samples, "keenetic_memory_cache_bytes", nil)
		Expect(ok).To(BeFalse())
	})
	It("parses actual DNS proxy text without resolving configured domains", func() {
		body := `{"proxy-status":[{"proxy-name":"System","proxy-stat":"# ndnproxy statistics file\nTotal incoming requests: 809\nProxy requests sent: 659\nCache hits ratio: 0.192 (155)\nMemory usage: 44.41K\nDNS Servers\nIp Port R.Sent A.Rcvd NX.Rcvd Med.Resp Avg.Resp Rank\n127.0.0.1 40500 2 2 0 40ms 40ms 10\n"}]}`
		samples, err := parseSamples("dns", []json.RawMessage{json.RawMessage(body)}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, ok := sampleValue(samples, "keenetic_dns_requests_total", map[string]string{"proxy": "System"})
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(809)))
		v, ok = sampleValue(samples, "keenetic_dns_cache_hits_total", nil)
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(155)))
		v, ok = sampleValue(samples, "keenetic_dns_upstream_response_duration_seconds", map[string]string{"server": "127.0.0.1", "port": "40500"})
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(0.04))
		for _, s := range samples {
			if s.name == "keenetic_dns_requests_total" {
				Expect(s.kind).To(Equal(prometheus.CounterValue))
			}
		}
	})
	It("counts hosts without emitting individual client identities", func() {
		samples, err := parseSamples("hosts", []json.RawMessage{json.RawMessage(`{"host":[{"mac":"private","active":true,"registered":true},{"active":false,"registered":false},{"link":"up"}]}`)}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, _ := sampleValue(samples, "keenetic_hosts", nil)
		Expect(v).To(Equal(float64(3)))
		v, _ = sampleValue(samples, "keenetic_hosts_active", nil)
		Expect(v).To(Equal(float64(2)))
		for _, s := range samples {
			Expect(s.labels).To(BeEmpty())
		}
	})
	It("separates configured static routes, DNS rules and per-group domain/IP entries", func() {
		samples, err := parseSamples("routing", []json.RawMessage{
			json.RawMessage(`[{"host":"192.0.2.1","interface":"Wireguard0"},{"network":"198.51.100.0","interface":"ISP"}]`),
			json.RawMessage(`{"vpn":{"include":[{"address":"example.test"},{"address":"192.0.2.0/24"}]}}`),
			json.RawMessage(`[{"group":"vpn","interface":"Wireguard0"}]`),
		}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, ok := sampleValue(samples, "keenetic_static_routes", map[string]string{"interface": "ISP"})
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(1)))
		v, ok = sampleValue(samples, "keenetic_dns_group_entries", map[string]string{"group": "vpn", "entry_type": "domain"})
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(1)))
		v, ok = sampleValue(samples, "keenetic_dns_group_entries", map[string]string{"group": "vpn", "entry_type": "ip"})
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(float64(1)))
	})
	DescribeTable("rejects absent or malformed data instead of publishing fabricated zeroes", func(collector, body string) {
		_, err := parseSamples(collector, []json.RawMessage{json.RawMessage(body)}, nil)
		Expect(err).To(HaveOccurred())
	}, Entry("empty system", "system", `{}`), Entry("empty DNS", "dns", `{"proxy-status":[]}`), Entry("malformed DNS text", "dns", `{"proxy-status":[{"proxy-name":"System","proxy-stat":"garbage"}]}`), Entry("missing host list", "hosts", `{}`), Entry("null interfaces", "interfaces", `null`), Entry("missing version", "info", `{}`))
	It("treats an explicitly empty host list as a real zero", func() {
		samples, err := parseSamples("hosts", []json.RawMessage{json.RawMessage(`{"host":[]}`)}, nil)
		Expect(err).NotTo(HaveOccurred())
		v, ok := sampleValue(samples, "keenetic_hosts", nil)
		Expect(ok).To(BeTrue())
		Expect(v).To(BeZero())
	})
})
