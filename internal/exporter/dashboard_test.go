package exporter

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Grafana dashboard freshness", func() {
	It("displays router-provided names, used RAM and physical speed", func() {
		b, err := os.ReadFile("../../grafana/keenetic.json")
		Expect(err).NotTo(HaveOccurred())
		var dashboard struct {
			Panels []struct {
				Targets []struct {
					Expr   string `json:"expr"`
					Legend string `json:"legendFormat"`
				} `json:"targets"`
			} `json:"panels"`
		}
		Expect(json.Unmarshal(b, &dashboard)).To(Succeed())
		var expressions []string
		proxyMetric := regexp.MustCompile(`keenetic_dns_(requests_total|forwarded_requests_total|cache_hits_total|upstream_)`)
		for _, p := range dashboard.Panels {
			for _, target := range p.Targets {
				expressions = append(expressions, target.Expr)
				if strings.Contains(target.Expr, "keenetic_interface_") || strings.Contains(target.Expr, "keenetic_static_routes{") || strings.Contains(target.Expr, "keenetic_dns_routes{") {
					Expect(target.Legend).To(ContainSubstring("{{interface_name}}"))
					Expect(target.Expr).To(ContainSubstring(`interface_name!=""`), "exclude historical series that cannot render a display name")
				}
				if proxyMetric.MatchString(target.Expr) {
					Expect(target.Expr).To(ContainSubstring(`proxy_name!=""`))
				}
				if strings.Contains(target.Legend, "{{proxy}}") {
					Fail("DNS legends must use proxy_name")
				}
				if strings.Contains(target.Expr, "keenetic_dns_upstream_") {
					Expect(target.Legend).To(ContainSubstring("{{upstream_name}}"))
					Expect(target.Expr).To(ContainSubstring(`upstream_name!=""`))
				}
			}
		}
		Expect(strings.Join(expressions, "\n")).To(And(ContainSubstring("keenetic_memory_used_bytes"), ContainSubstring("keenetic_interface_speed_bits_per_second"), ContainSubstring("keenetic_static_routes{"), ContainSubstring("keenetic_dns_routes{")))
	})
	It("uses instant queries for current-value stat panels so expired series are not resurrected from history", func() {
		b, err := os.ReadFile("../../grafana/keenetic.json")
		Expect(err).NotTo(HaveOccurred())
		var dashboard struct {
			Panels []struct {
				Type    string `json:"type"`
				Targets []struct {
					Instant bool `json:"instant"`
					Range   bool `json:"range"`
				} `json:"targets"`
			} `json:"panels"`
		}
		Expect(json.Unmarshal(b, &dashboard)).To(Succeed())
		for _, p := range dashboard.Panels {
			if p.Type == "stat" {
				for _, target := range p.Targets {
					Expect(target.Instant).To(BeTrue())
					Expect(target.Range).To(BeFalse())
				}
			}
		}
	})
})
