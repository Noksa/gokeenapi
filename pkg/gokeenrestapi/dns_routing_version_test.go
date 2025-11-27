package gokeenrestapi

import (
	"testing"

	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/stretchr/testify/assert"
)

func TestCheckDnsRoutingSupport_SupportedVersion(t *testing.T) {
	// Setup: Set version to 5.0.1 (minimum required)
	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = gokeenrestapimodels.Version{
			Title: "5.0.1",
		}
	})

	err := DnsRouting.CheckDnsRoutingSupport()
	assert.NoError(t, err, "Version 5.0.1 should be supported")
}

func TestCheckDnsRoutingSupport_HigherVersion(t *testing.T) {
	// Setup: Set version to 5.1.0 (higher than minimum)
	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = gokeenrestapimodels.Version{
			Title: "5.1.0",
		}
	})

	err := DnsRouting.CheckDnsRoutingSupport()
	assert.NoError(t, err, "Version 5.1.0 should be supported")
}

func TestCheckDnsRoutingSupport_UnsupportedVersion(t *testing.T) {
	// Setup: Set version to 4.3.6.3 (below minimum)
	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = gokeenrestapimodels.Version{
			Title: "4.3.6.3",
		}
	})

	err := DnsRouting.CheckDnsRoutingSupport()
	assert.Error(t, err, "Version 4.3.6.3 should not be supported")
	assert.Contains(t, err.Error(), "DNS-routing requires Keenetic firmware version 5.0.1 or higher")
	assert.Contains(t, err.Error(), "4.3.6.3")
}

func TestCheckDnsRoutingSupport_EdgeCaseJustBelow(t *testing.T) {
	// Setup: Set version to 5.0.0 (just below minimum)
	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = gokeenrestapimodels.Version{
			Title: "5.0.0",
		}
	})

	err := DnsRouting.CheckDnsRoutingSupport()
	assert.Error(t, err, "Version 5.0.0 should not be supported")
	assert.Contains(t, err.Error(), "DNS-routing requires Keenetic firmware version 5.0.1 or higher")
}

func TestCheckDnsRoutingSupport_MissingVersion(t *testing.T) {
	// Setup: Set empty version
	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = gokeenrestapimodels.Version{
			Title: "",
		}
	})

	err := DnsRouting.CheckDnsRoutingSupport()
	assert.Error(t, err, "Empty version should return error")
	assert.Contains(t, err.Error(), "router version information not available")
}
