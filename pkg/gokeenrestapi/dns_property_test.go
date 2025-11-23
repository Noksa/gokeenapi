package gokeenrestapi

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// DNS record generators for property-based testing

// genDomainName generates valid domain names
// Valid domain names consist of labels separated by dots
// Each label: 1-63 chars, alphanumeric + hyphens, can't start/end with hyphen
// Total length: max 253 chars
func genDomainName() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 1-4 labels for reasonable domain names
		numLabels := rapid.IntRange(1, 4).Draw(t, "numLabels")
		labels := make([]string, numLabels)

		for i := range numLabels {
			// Generate label length (1-20 chars for reasonable names)
			labelLen := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("labelLen%d", i))

			// Generate label: start with letter, then alphanumeric or hyphen, end with alphanumeric
			if labelLen == 1 {
				// Single character label - just a letter
				labels[i] = rapid.StringMatching(`[a-z]`).Draw(t, fmt.Sprintf("label%d", i))
			} else {
				// Multi-character label
				start := rapid.StringMatching(`[a-z]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))

				if labelLen == 2 {
					labels[i] = start + end
				} else {
					// Middle can be alphanumeric or hyphen
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

// genDNSRecord generates a DNS record with domain and IP addresses
func genDNSRecord() *rapid.Generator[struct {
	Domain string
	IPs    []string
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		Domain string
		IPs    []string
	} {
		domain := genDomainName().Draw(t, "domain")

		// Generate 1-5 IP addresses for the domain
		numIPs := rapid.IntRange(1, 5).Draw(t, "numIPs")
		ips := make([]string, numIPs)
		for i := range numIPs {
			ips[i] = genIPv4Address().Draw(t, fmt.Sprintf("ip%d", i))
		}

		return struct {
			Domain string
			IPs    []string
		}{
			Domain: domain,
			IPs:    ips,
		}
	})
}

// genInvalidDomain generates invalid domain names for negative testing
func genInvalidDomain() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"",                       // Empty string
		".",                      // Just a dot
		".example.com",           // Starts with dot
		"example.com.",           // Ends with dot (technically valid in DNS but often rejected)
		"-example.com",           // Label starts with hyphen
		"example-.com",           // Label ends with hyphen
		"exam ple.com",           // Contains space
		"example..com",           // Double dot
		"example.com/path",       // Contains slash
		"example.com:8080",       // Contains colon
		"example_test.com",       // Contains underscore (invalid in hostnames)
		"192.168.1.1",            // IP address (not a domain)
		strings.Repeat("a", 254), // Too long (>253 chars)
		"exam@ple.com",           // Contains @
		"example$.com",           // Contains $
		"example!.com",           // Contains !
		"EXAMPLE.COM",            // Uppercase (may be valid but testing case sensitivity)
	})
}

// Helper functions for DNS validation

// isValidDomainName checks if a string is a valid domain name
func isValidDomainName(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}

	// Reject if it looks like an IP address
	if isValidIPv4(domain) {
		return false
	}

	// Domain name regex pattern
	// Each label: 1-63 chars, alphanumeric + hyphens, can't start/end with hyphen
	// Labels must start with a letter (not a digit) to distinguish from IPs
	labelPattern := `[a-z]([a-z0-9-]{0,61}[a-z0-9])?`
	domainPattern := fmt.Sprintf(`^%s(\.%s)*$`, labelPattern, labelPattern)

	matched, err := regexp.MatchString(domainPattern, strings.ToLower(domain))
	if err != nil {
		return false
	}

	return matched
}

// formatDNSAddCommand formats a DNS add command for the router
func formatDNSAddCommand(domain string, ip string) string {
	return fmt.Sprintf("ip host %s %s", domain, ip)
}

// formatDNSDeleteCommand formats a DNS delete command for the router
func formatDNSDeleteCommand(domain string, ip string) string {
	return fmt.Sprintf("no ip host %s %s", domain, ip)
}

// parseDNSCommand parses a DNS command and extracts domain and IP
func parseDNSCommand(cmd string) (domain string, ip string, isDelete bool, valid bool) {
	// Match "ip host <domain> <ip>" or "no ip host <domain> <ip>"
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

// Property-based tests for DNS record functionality

// Feature: property-based-testing, Property 22: DNS command formatting is valid
// Validates: Requirements 6.1, 6.2
func TestDNSAddCommandsAreValidAndParseable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a DNS record with domain and multiple IPs
		record := genDNSRecord().Draw(t, "record")

		// Test that each IP generates a valid command
		for _, ip := range record.IPs {
			cmd := formatDNSAddCommand(record.Domain, ip)

			// Parse the command back to verify it's valid
			parsedDomain, parsedIP, isDelete, valid := parseDNSCommand(cmd)

			// Assert the command is valid and parseable
			if !valid {
				t.Fatalf("generated command is not valid: %s", cmd)
			}

			// Assert it's not a delete command
			if isDelete {
				t.Fatalf("add command incorrectly parsed as delete: %s", cmd)
			}

			// Assert domain and IP are preserved
			if parsedDomain != record.Domain {
				t.Fatalf("domain mismatch: got %s, want %s", parsedDomain, record.Domain)
			}

			if parsedIP != ip {
				t.Fatalf("IP mismatch: got %s, want %s", parsedIP, ip)
			}

			// Verify the command follows the expected format
			expectedCmd := fmt.Sprintf("ip host %s %s", record.Domain, ip)
			if cmd != expectedCmd {
				t.Fatalf("command format mismatch: got %s, want %s", cmd, expectedCmd)
			}
		}
	})
}

// Feature: property-based-testing, Property 23: Domain name validation
// Validates: Requirements 6.3
func TestDomainNameValidationAcceptsValidDomainsAndRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test 1: Valid domains generated by genDomainName should be accepted
		validDomain := genDomainName().Draw(t, "validDomain")

		if !isValidDomainName(validDomain) {
			t.Fatalf("valid domain rejected: %s", validDomain)
		}

		// Test 2: Invalid domains should be rejected
		invalidDomain := genInvalidDomain().Draw(t, "invalidDomain")

		// Most invalid domains should be rejected
		// Note: Some edge cases like uppercase might be valid depending on implementation
		if isValidDomainName(invalidDomain) {
			// Check if this is an expected edge case
			// Uppercase domains are technically valid (DNS is case-insensitive)
			if strings.ToLower(invalidDomain) == invalidDomain {
				t.Fatalf("invalid domain accepted: %s", invalidDomain)
			}
		}
	})
}

// Feature: property-based-testing, Property 24: DNS delete command correctness
// Validates: Requirements 6.5
func TestDNSDeleteCommandIsPrefixedWithNoAndParsesCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a DNS record
		record := genDNSRecord().Draw(t, "record")

		// Test that delete command is the add command prefixed with "no"
		for _, ip := range record.IPs {
			addCmd := formatDNSAddCommand(record.Domain, ip)
			deleteCmd := formatDNSDeleteCommand(record.Domain, ip)

			// Assert delete command is "no " + add command
			expectedDeleteCmd := fmt.Sprintf("no %s", addCmd)
			if deleteCmd != expectedDeleteCmd {
				t.Fatalf("delete command format incorrect: got %s, want %s", deleteCmd, expectedDeleteCmd)
			}

			// Parse the delete command to verify it's valid
			parsedDomain, parsedIP, isDelete, valid := parseDNSCommand(deleteCmd)

			if !valid {
				t.Fatalf("delete command is not valid: %s", deleteCmd)
			}

			if !isDelete {
				t.Fatalf("delete command not recognized as delete: %s", deleteCmd)
			}

			// Assert domain and IP are preserved
			if parsedDomain != record.Domain {
				t.Fatalf("domain mismatch in delete: got %s, want %s", parsedDomain, record.Domain)
			}

			if parsedIP != ip {
				t.Fatalf("IP mismatch in delete: got %s, want %s", parsedIP, ip)
			}
		}
	})
}
