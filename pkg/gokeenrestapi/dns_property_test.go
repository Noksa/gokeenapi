package gokeenrestapi

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// DNS record generators for property-based testing

func genDomainName() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		numLabels := rapid.IntRange(1, 4).Draw(t, "numLabels")
		labels := make([]string, numLabels)

		for i := range numLabels {
			labelLen := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("labelLen%d", i))

			if labelLen == 1 {
				labels[i] = rapid.StringMatching(`[a-z]`).Draw(t, fmt.Sprintf("label%d", i))
			} else {
				start := rapid.StringMatching(`[a-z]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))

				if labelLen == 2 {
					labels[i] = start + end
				} else {
					middleLen := labelLen - 2
					middle := rapid.StringMatching(fmt.Sprintf(`[a-z0-9-]{%d}`, middleLen)).
						Draw(t, fmt.Sprintf("labelMiddle%d", i))
					labels[i] = start + middle + end
				}
			}
		}
		return strings.Join(labels, ".")
	})
}

func genDNSRecord() *rapid.Generator[struct {
	Domain string
	IPs    []string
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		Domain string
		IPs    []string
	} {
		domain := genDomainName().Draw(t, "domain")
		numIPs := rapid.IntRange(1, 5).Draw(t, "numIPs")
		ips := make([]string, numIPs)
		for i := range numIPs {
			ips[i] = genIPv4Address().Draw(t, fmt.Sprintf("ip%d", i))
		}
		return struct {
			Domain string
			IPs    []string
		}{Domain: domain, IPs: ips}
	})
}

func genInvalidDomain() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"", ".", ".example.com", "example.com.", "-example.com",
		"example-.com", "exam ple.com", "example..com", "example.com/path",
		"example.com:8080", "example_test.com", "192.168.1.1",
		strings.Repeat("a", 254), "exam@ple.com", "example$.com",
		"example!.com", "EXAMPLE.COM",
	})
}

func isValidDomainName(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	if isValidIPv4(domain) {
		return false
	}
	labelPattern := `[a-z]([a-z0-9-]{0,61}[a-z0-9])?`
	domainPattern := fmt.Sprintf(`^%s(\.%s)*$`, labelPattern, labelPattern)
	matched, err := regexp.MatchString(domainPattern, strings.ToLower(domain))
	if err != nil {
		return false
	}
	return matched
}

func formatDNSAddCommand(domain string, ip string) string {
	return fmt.Sprintf("ip host %s %s", domain, ip)
}

func formatDNSDeleteCommand(domain string, ip string) string {
	return fmt.Sprintf("no ip host %s %s", domain, ip)
}

func parseDNSCommand(cmd string) (domain string, ip string, isDelete bool, valid bool) {
	deletePattern := regexp.MustCompile(`^no\s+ip\s+host\s+(\S+)\s+(\S+)$`)
	addPattern := regexp.MustCompile(`^ip\s+host\s+(\S+)\s+(\S+)$`)

	if matches := deletePattern.FindStringSubmatch(cmd); matches != nil {
		return matches[1], matches[2], true, true
	}
	if matches := addPattern.FindStringSubmatch(cmd); matches != nil {
		return matches[1], matches[2], false, true
	}
	return "", "", false, false
}

var _ = Describe("Property: DNS Records", func() {
	It("should produce valid and parseable add commands", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			record := genDNSRecord().Draw(t, "record")

			for _, ip := range record.IPs {
				cmd := formatDNSAddCommand(record.Domain, ip)
				parsedDomain, parsedIP, isDelete, valid := parseDNSCommand(cmd)

				Expect(valid).To(BeTrue(), "command not valid: %s", cmd)
				Expect(isDelete).To(BeFalse(), "add command parsed as delete: %s", cmd)
				Expect(parsedDomain).To(Equal(record.Domain))
				Expect(parsedIP).To(Equal(ip))
				Expect(cmd).To(Equal(fmt.Sprintf("ip host %s %s", record.Domain, ip)))
			}
		})
	})

	It("should validate domain names correctly", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			validDomain := genDomainName().Draw(t, "validDomain")
			Expect(isValidDomainName(validDomain)).To(BeTrue(), "valid domain rejected: %s", validDomain)

			invalidDomain := genInvalidDomain().Draw(t, "invalidDomain")
			if isValidDomainName(invalidDomain) {
				// Uppercase domains are technically valid
				Expect(strings.ToLower(invalidDomain)).NotTo(Equal(invalidDomain),
					"invalid domain accepted: %s", invalidDomain)
			}
		})
	})

	It("should produce correct delete commands", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			record := genDNSRecord().Draw(t, "record")

			for _, ip := range record.IPs {
				addCmd := formatDNSAddCommand(record.Domain, ip)
				deleteCmd := formatDNSDeleteCommand(record.Domain, ip)

				Expect(deleteCmd).To(Equal(fmt.Sprintf("no %s", addCmd)))

				parsedDomain, parsedIP, isDelete, valid := parseDNSCommand(deleteCmd)
				Expect(valid).To(BeTrue(), "delete command not valid: %s", deleteCmd)
				Expect(isDelete).To(BeTrue(), "delete command not recognized: %s", deleteCmd)
				Expect(parsedDomain).To(Equal(record.Domain))
				Expect(parsedIP).To(Equal(ip))
			}
		})
	})
})
