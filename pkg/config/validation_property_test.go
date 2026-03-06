package config

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// genMalformedDomain generates malformed domain names
func genMalformedDomain() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		choice := rapid.IntRange(0, 8).Draw(t, "choice")
		switch choice {
		case 0:
			return rapid.StringMatching(`-[a-z0-9-]{2,10}\.[a-z]{2,5}`).Draw(t, "domain")
		case 1:
			return rapid.StringMatching(`[a-z][a-z0-9-]{2,10}-\.[a-z]{2,5}`).Draw(t, "domain")
		case 2:
			return rapid.StringMatching(`[a-z][a-z0-9]{1,5}\.\.[a-z]{2,5}`).Draw(t, "domain")
		case 3:
			return rapid.StringMatching(`\.[a-z][a-z0-9]{2,10}\.[a-z]{2,5}`).Draw(t, "domain")
		case 4:
			return rapid.StringMatching(`[a-z][a-z0-9]{1,5}[@#$%][a-z0-9]{1,5}\.[a-z]{2,5}`).Draw(t, "domain")
		case 5:
			return rapid.StringMatching(`[a-z][a-z0-9]{1,5} [a-z0-9]{1,5}\.[a-z]{2,5}`).Draw(t, "domain")
		case 6:
			return rapid.StringMatching(`[a-z][a-z0-9-]{5,15}`).Draw(t, "domain")
		case 7:
			return rapid.StringMatching(`[a-z][a-z0-9]{2,10}\.123`).Draw(t, "domain")
		case 8:
			longLabel := rapid.StringMatching(`[a-z][a-z0-9]{64,80}`).Draw(t, "longLabel")
			return longLabel + ".com"
		default:
			return "invalid..domain"
		}
	})
}

// genMalformedIP generates malformed IP addresses
func genMalformedIP() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		choice := rapid.IntRange(0, 8).Draw(t, "choice")
		switch choice {
		case 0:
			octet1 := rapid.IntRange(256, 999).Draw(t, "octet1")
			return fmt.Sprintf("%d.1.1.1", octet1)
		case 1:
			return "-1.1.1.1"
		case 2:
			return rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "ip")
		case 3:
			return rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "ip")
		case 4:
			return "01.02.03.04"
		case 5:
			return rapid.StringMatching(`[a-z]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "ip")
		case 6:
			return "192..1.1"
		case 7:
			return "192.168. 1.1"
		case 8:
			return "192.168.1.1!"
		default:
			return "999.999.999.999"
		}
	})
}

var _ = Describe("Property: DNS Routing Group Validation", func() {
	It("should reject empty or whitespace-only group names", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			invalidName := genWhitespaceString().Draw(t, "invalidName")
			validGroup := genDnsRoutingGroup().Draw(t, "validGroup")

			invalidGroup := DnsRoutingGroup{
				Name:        invalidName,
				DomainFile:  validGroup.DomainFile,
				InterfaceID: validGroup.InterfaceID,
			}

			err := ValidateDnsRoutingGroups([]DnsRoutingGroup{invalidGroup})
			Expect(err).To(HaveOccurred(),
				"validation should reject empty or whitespace-only group name %q", invalidName)
			Expect(err.Error()).NotTo(BeEmpty(), "validation error should have a descriptive message")
		})
	})

	It("should reject empty domain lists", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			validGroup := genDnsRoutingGroup().Draw(t, "validGroup")

			invalidGroup := DnsRoutingGroup{
				Name:        validGroup.Name,
				DomainFile:  []string{},
				DomainURL:   []string{},
				InterfaceID: validGroup.InterfaceID,
			}

			err := ValidateDnsRoutingGroups([]DnsRoutingGroup{invalidGroup})
			Expect(err).To(HaveOccurred(),
				"validation should reject DNS routing group %q with no domain sources", invalidGroup.Name)
			Expect(err.Error()).NotTo(BeEmpty(), "validation error should have a descriptive message")
		})
	})
})

var _ = Describe("Property: Malformed Domains and IPs Validation", func() {
	It("should reject malformed domains and IP addresses", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			testDomain := rapid.Bool().Draw(t, "testDomain")

			var invalidDomain string
			if testDomain {
				invalidDomain = genMalformedDomain().Draw(t, "malformedDomain")
			} else {
				invalidDomain = genMalformedIP().Draw(t, "malformedIP")
			}

			err := ValidateDomainList([]string{invalidDomain}, "test-group")
			Expect(err).To(HaveOccurred(),
				"validation should reject malformed domain/IP %q", invalidDomain)
			Expect(err.Error()).NotTo(BeEmpty(), "validation error should have a descriptive message")
		})
	})
})
