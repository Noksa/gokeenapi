package gokeenrestapi

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// Domain line generators for property-based testing

// genValidDomain generates valid domain names for testing
func genValidDomain() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		numLabels := rapid.IntRange(2, 4).Draw(t, "numLabels")
		labels := make([]string, numLabels)

		for i := range numLabels {
			labelLen := rapid.IntRange(1, 15).Draw(t, fmt.Sprintf("labelLen%d", i))

			switch {
			case labelLen == 1:
				labels[i] = rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("label%d", i))
			case labelLen == 2:
				start := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))
				labels[i] = start + end
			default:
				start := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))
				middleLen := labelLen - 2
				middle := rapid.StringMatching(fmt.Sprintf(`[a-z0-9]{%d}`, middleLen)).
					Draw(t, fmt.Sprintf("labelMiddle%d", i))
				labels[i] = start + middle + end
			}
		}

		return strings.Join(labels, ".")
	})
}

// genV2flyPrefix generates v2fly domain-list-community prefixes
func genV2flyPrefix() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"full:",
		"regexp:",
		"domain:",
		"keyword:",
		"include:",
	})
}

// genDomainLineWithPrefix generates domain lines with v2fly prefixes
func genDomainLineWithPrefix() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		prefix := genV2flyPrefix().Draw(t, "prefix")
		domain := genValidDomain().Draw(t, "domain")
		return prefix + domain
	})
}

// genDomainLineWithAttributes generates domain lines with attributes (e.g., "@cn")
func genDomainLineWithAttributes() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		domain := genValidDomain().Draw(t, "domain")
		numAttrs := rapid.IntRange(1, 3).Draw(t, "numAttrs")
		attrs := make([]string, numAttrs)
		for i := range numAttrs {
			attrs[i] = rapid.StringMatching(`@[a-z]{2}`).Draw(t, fmt.Sprintf("attr%d", i))
		}
		return domain + " " + strings.Join(attrs, " ")
	})
}

// genComment generates comment lines
func genComment() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		comment := rapid.String().Draw(t, "comment")
		return "# " + comment
	})
}

// genEmptyLine generates empty or whitespace-only lines
func genEmptyLine() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"", " ", "  ", "\t", " \t "})
}

// genInvalidDomainLine generates invalid domain lines
func genInvalidDomainLine() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"youtube",
		"instagram",
		"@cn",
		"regexp:.*\\.com$",
		"..example.com",
		"-example.com",
		"example-.com",
		"exam ple.com",
		"example..com",
	})
}

// genMixedDomainLines generates a mix of valid, invalid, comment, and empty lines
func genMixedDomainLines() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		numLines := rapid.IntRange(5, 20).Draw(t, "numLines")
		lines := make([]string, numLines)

		for i := range numLines {
			switch rapid.IntRange(0, 5).Draw(t, fmt.Sprintf("lineType%d", i)) {
			case 0:
				lines[i] = genValidDomain().Draw(t, fmt.Sprintf("validDomain%d", i))
			case 1:
				lines[i] = genDomainLineWithPrefix().Draw(t, fmt.Sprintf("prefixDomain%d", i))
			case 2:
				lines[i] = genDomainLineWithAttributes().Draw(t, fmt.Sprintf("attrDomain%d", i))
			case 3:
				lines[i] = genComment().Draw(t, fmt.Sprintf("comment%d", i))
			case 4:
				lines[i] = genEmptyLine().Draw(t, fmt.Sprintf("empty%d", i))
			case 5:
				lines[i] = genInvalidDomainLine().Draw(t, fmt.Sprintf("invalid%d", i))
			}
		}

		return lines
	})
}

var _ = Describe("Property: Domain Line Parsing", func() {
	It("should never return empty domains", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			lines := genMixedDomainLines().Draw(t, "lines")
			domains, _ := parseDomainLines(lines, "test")

			for _, domain := range domains {
				Expect(domain).NotTo(BeEmpty(), "parseDomainLines returned empty domain")
			}
		})
	})

	It("should return only IDNA-valid domains", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			lines := genMixedDomainLines().Draw(t, "lines")
			domains, _ := parseDomainLines(lines, "test")

			for _, domain := range domains {
				valid, reason := validateDomainWithIDNA(domain)
				Expect(valid).To(BeTrue(),
					"parseDomainLines returned invalid domain '%s': %s", domain, reason)
			}
		})
	})

	It("should skip comment lines", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			numLines := rapid.IntRange(5, 15).Draw(t, "numLines")
			lines := make([]string, numLines)

			for i := range numLines {
				if rapid.Bool().Draw(t, fmt.Sprintf("isComment%d", i)) {
					lines[i] = genComment().Draw(t, fmt.Sprintf("comment%d", i))
				} else {
					lines[i] = genValidDomain().Draw(t, fmt.Sprintf("domain%d", i))
				}
			}

			domains, _ := parseDomainLines(lines, "test")

			for _, domain := range domains {
				Expect(domain).NotTo(HavePrefix("#"),
					"parseDomainLines returned comment as domain: %s", domain)
			}
		})
	})

	It("should strip v2fly prefixes", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			prefix := genV2flyPrefix().Draw(t, "prefix")
			domain := genValidDomain().Draw(t, "domain")
			line := prefix + domain

			domains, _ := parseDomainLines([]string{line}, "test")

			Expect(domains).To(HaveLen(1))
			Expect(domains[0]).To(Equal(domain),
				"prefix not stripped correctly: got %s, want %s", domains[0], domain)
			Expect(domains[0]).NotTo(ContainSubstring(":"),
				"domain still contains colon (prefix not removed): %s", domains[0])
		})
	})

	It("should strip attributes from domain lines", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			domain := genValidDomain().Draw(t, "domain")
			attribute := rapid.StringMatching(`@[a-z]{2}`).Draw(t, "attribute")
			line := domain + " " + attribute

			domains, _ := parseDomainLines([]string{line}, "test")

			if len(domains) == 0 {
				return // domain was invalid, acceptable
			}

			Expect(domains).To(HaveLen(1))
			Expect(domains[0]).To(Equal(domain),
				"attribute not stripped correctly: got %s, want %s", domains[0], domain)
			Expect(domains[0]).NotTo(ContainSubstring("@"),
				"domain still contains @ (attribute not removed): %s", domains[0])
		})
	})

	It("should count skipped lines correctly", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			numValid := rapid.IntRange(1, 10).Draw(t, "numValid")
			numComments := rapid.IntRange(0, 5).Draw(t, "numComments")
			numEmpty := rapid.IntRange(0, 5).Draw(t, "numEmpty")
			numInvalid := rapid.IntRange(1, 5).Draw(t, "numInvalid")

			lines := make([]string, 0, numValid+numComments+numEmpty+numInvalid)

			for i := range numValid {
				lines = append(lines, genValidDomain().Draw(t, fmt.Sprintf("valid%d", i)))
			}
			for i := range numInvalid {
				bareName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, fmt.Sprintf("invalid%d", i))
				lines = append(lines, bareName)
			}
			for i := range numComments {
				lines = append(lines, genComment().Draw(t, fmt.Sprintf("comment%d", i)))
			}
			for i := range numEmpty {
				lines = append(lines, genEmptyLine().Draw(t, fmt.Sprintf("empty%d", i)))
			}

			domains, skipped := parseDomainLines(lines, "test")

			totalProcessed := len(domains) + skipped
			expectedProcessed := numValid + numInvalid
			Expect(totalProcessed).To(Equal(expectedProcessed),
				"count mismatch: got %d valid + %d skipped = %d, want %d",
				len(domains), skipped, totalProcessed, expectedProcessed)

			Expect(skipped).To(Equal(numInvalid),
				"skipped count mismatch: got %d, expected %d", skipped, numInvalid)
		})
	})

	It("should be idempotent", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			lines := genMixedDomainLines().Draw(t, "lines")

			domains1, skipped1 := parseDomainLines(lines, "test")
			domains2, skipped2 := parseDomainLines(lines, "test")

			Expect(domains2).To(HaveLen(len(domains1)))
			Expect(skipped2).To(Equal(skipped1))

			for i := range domains1 {
				Expect(domains2[i]).To(Equal(domains1[i]),
					"domain at index %d differs", i)
			}
		})
	})

	It("should preserve order of valid domains", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			numDomains := rapid.IntRange(3, 10).Draw(t, "numDomains")
			expectedDomains := make([]string, 0, numDomains)
			seen := make(map[string]bool)

			for len(expectedDomains) < numDomains {
				domain := genValidDomain().Draw(t, fmt.Sprintf("domain%d", len(expectedDomains)))
				if !seen[domain] {
					seen[domain] = true
					expectedDomains = append(expectedDomains, domain)
				}
			}

			domains, _ := parseDomainLines(expectedDomains, "test")

			Expect(domains).To(HaveLen(len(expectedDomains)),
				"domain count mismatch: got %d, want %d", len(domains), len(expectedDomains))

			for i := range domains {
				Expect(domains[i]).To(Equal(expectedDomains[i]),
					"order not preserved at index %d: got %s, want %s", i, domains[i], expectedDomains[i])
			}
		})
	})
})

var _ = Describe("Property: IDNA Domain Validation", func() {
	It("should reject bare domains without TLD", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			bareDomain := rapid.StringMatching(`[a-z]{3,15}`).Draw(t, "bareDomain")

			valid, reason := validateDomainWithIDNA(bareDomain)

			Expect(valid).To(BeFalse(), "bare domain incorrectly accepted: %s", bareDomain)
			Expect(reason).To(SatisfyAny(ContainSubstring("TLD"), ContainSubstring("dot")),
				"rejection reason doesn't mention TLD: %s", reason)
		})
	})

	It("should accept properly formatted domains", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			domain := genValidDomain().Draw(t, "domain")

			valid, reason := validateDomainWithIDNA(domain)

			Expect(valid).To(BeTrue(), "valid domain rejected: %s (reason: %s)", domain, reason)
		})
	})

	It("should reject empty string", func() {
		valid, reason := validateDomainWithIDNA("")

		Expect(valid).To(BeFalse(), "empty string incorrectly accepted as domain")
		Expect(reason).To(ContainSubstring("empty"))
	})
})
