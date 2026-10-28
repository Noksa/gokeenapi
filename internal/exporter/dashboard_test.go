package exporter

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Grafana dashboard freshness", func() {
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
