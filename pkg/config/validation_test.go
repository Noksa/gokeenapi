package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValidateDnsRoutingGroups", func() {
	It("should accept valid groups", func() {
		groups := []DnsRoutingGroup{
			{Name: "group1", DomainFile: []string{"/path/to/domains.txt"}, InterfaceID: "Wireguard0"},
			{Name: "group2", DomainURL: []string{"https://example.com/domains.txt"}, InterfaceID: "Wireguard1"},
		}
		Expect(ValidateDnsRoutingGroups(groups)).To(Succeed())
	})

	It("should reject empty name", func() {
		groups := []DnsRoutingGroup{{Name: "", DomainFile: []string{"/path/to/domains.txt"}, InterfaceID: "Wireguard0"}}
		err := ValidateDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("name cannot be empty"))
	})

	It("should reject whitespace-only name", func() {
		groups := []DnsRoutingGroup{{Name: "   \t\n  ", DomainFile: []string{"/path/to/domains.txt"}, InterfaceID: "Wireguard0"}}
		err := ValidateDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot contain only whitespace"))
	})

	It("should reject duplicate names", func() {
		groups := []DnsRoutingGroup{
			{Name: "youtube", DomainFile: []string{"/path/to/youtube.txt"}, InterfaceID: "Wireguard0"},
			{Name: "instagram", DomainURL: []string{"https://example.com/instagram.txt"}, InterfaceID: "Wireguard0"},
			{Name: "youtube", DomainFile: []string{"/path/to/youtube2.txt"}, InterfaceID: "Wireguard1"},
		}
		err := ValidateDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate DNS routing group name"))
		Expect(err.Error()).To(ContainSubstring("youtube"))
	})

	It("should reject groups with no domain sources", func() {
		groups := []DnsRoutingGroup{{Name: "group1", InterfaceID: "Wireguard0"}}
		err := ValidateDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must contain at least one domain-file or domain-url"))
	})

	It("should reject empty interface ID", func() {
		groups := []DnsRoutingGroup{{Name: "group1", DomainFile: []string{"/path/to/domains.txt"}, InterfaceID: ""}}
		err := ValidateDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("interface ID cannot be empty"))
	})

	It("should accept empty list", func() {
		Expect(ValidateDnsRoutingGroups([]DnsRoutingGroup{})).To(Succeed())
	})

	It("should accept groups with both domain-file and domain-url", func() {
		groups := []DnsRoutingGroup{
			{Name: "group1", DomainFile: []string{"/path/to/domains.txt"}, DomainURL: []string{"https://example.com/domains.txt"}, InterfaceID: "Wireguard0"},
		}
		Expect(ValidateDnsRoutingGroups(groups)).To(Succeed())
	})
})

var _ = Describe("ValidateDomainList", func() {
	It("should accept valid domains", func() {
		Expect(ValidateDomainList([]string{"example.com", "sub.example.com", "another-domain.org"}, "testgroup")).To(Succeed())
	})

	It("should accept valid IPs", func() {
		Expect(ValidateDomainList([]string{"192.168.1.1", "10.0.0.1", "8.8.8.8"}, "testgroup")).To(Succeed())
	})

	It("should accept mixed domains and IPs", func() {
		Expect(ValidateDomainList([]string{"example.com", "192.168.1.1", "sub.example.org", "10.0.0.1"}, "testgroup")).To(Succeed())
	})

	It("should reject empty domain", func() {
		err := ValidateDomainList([]string{"example.com", "", "another.com"}, "testgroup")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot be empty"))
	})

	It("should reject whitespace-only domain", func() {
		err := ValidateDomainList([]string{"example.com", "   \t  ", "another.com"}, "testgroup")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot contain only whitespace"))
	})

	It("should reject invalid domain", func() {
		err := ValidateDomainList([]string{"example.com", "invalid domain with spaces", "another.com"}, "testgroup")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid domain or IP address"))
	})

	It("should reject invalid IP", func() {
		err := ValidateDomainList([]string{"example.com", "999.999.999.999", "another.com"}, "testgroup")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid domain or IP address"))
	})

	It("should accept empty list", func() {
		Expect(ValidateDomainList([]string{}, "testgroup")).To(Succeed())
	})
})
