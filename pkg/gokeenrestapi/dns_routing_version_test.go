package gokeenrestapi

import (
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckDnsRoutingSupport", func() {
	It("should support version 5.0.1", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.0.1"}
		})
		Expect(DnsRouting.CheckDnsRoutingSupport()).To(Succeed())
	})

	It("should support higher version", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.1.0"}
		})
		Expect(DnsRouting.CheckDnsRoutingSupport()).To(Succeed())
	})

	It("should reject unsupported version", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "4.3.6.3"}
		})
		err := DnsRouting.CheckDnsRoutingSupport()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DNS-routing requires Keenetic firmware version 5.0.1 or higher"))
		Expect(err.Error()).To(ContainSubstring("4.3.6.3"))
	})

	It("should reject version just below minimum", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.0.0"}
		})
		err := DnsRouting.CheckDnsRoutingSupport()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DNS-routing requires Keenetic firmware version 5.0.1 or higher"))
	})

	It("should reject missing version", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: ""}
		})
		err := DnsRouting.CheckDnsRoutingSupport()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("router version information not available"))
	})
})
