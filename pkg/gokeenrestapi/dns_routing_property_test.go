package gokeenrestapi

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Domain line generators for property-based testing

// genValidDomain generates valid domain names for testing
// Follows IDNA rules: labels can't start/end with hyphen, can't be all hyphens
func genValidDomain() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 2-4 labels for realistic domain names (must have TLD)
		numLabels := rapid.IntRange(2, 4).Draw(t, "numLabels")
		labels := make([]string, numLabels)

		for i := range numLabels {
			// Generate label length (1-15 chars for reasonable names)
			labelLen := rapid.IntRange(1, 15).Draw(t, fmt.Sprintf("labelLen%d", i))

			if labelLen == 1 {
				// Single character label - alphanumeric only (no hyphen)
				labels[i] = rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("label%d", i))
			} else if labelLen == 2 {
				// Two character label - both alphanumeric (no hyphen at start/end)
				start := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))
				labels[i] = start + end
			} else {
				// Multi-character label: start and end with alphanumeric
				// Middle can be alphanumeric or single hyphens (not consecutive)
				start := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelStart%d", i))
				end := rapid.StringMatching(`[a-z0-9]`).Draw(t, fmt.Sprintf("labelEnd%d", i))

				// Generate middle part without consecutive hyphens
				middleLen := labelLen - 2
				// Use only alphanumeric for simplicity (IDNA is strict about hyphens)
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
		// Generate 1-3 attributes
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
	return rapid.SampledFrom([]string{
		"",
		" ",
		"  ",
		"\t",
		" \t ",
	})
}

// genInvalidDomainLine generates invalid domain lines
func genInvalidDomainLine() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"youtube",          // No TLD
		"instagram",        // No TLD
		"@cn",              // Just an attribute
		"regexp:.*\\.com$", // Regexp pattern (after prefix removal, still invalid)
		"..example.com",    // Double dot
		"-example.com",     // Starts with hyphen
		"example-.com",     // Label ends with hyphen
		"exam ple.com",     // Space in domain
		"example..com",     // Consecutive dots
	})
}

// genMixedDomainLines generates a mix of valid, invalid, comment, and empty lines
func genMixedDomainLines() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		// Generate 5-20 lines
		numLines := rapid.IntRange(5, 20).Draw(t, "numLines")
		lines := make([]string, numLines)

		for i := range numLines {
			// Choose line type
			lineType := rapid.IntRange(0, 5).Draw(t, fmt.Sprintf("lineType%d", i))

			switch lineType {
			case 0:
				// Valid domain
				lines[i] = genValidDomain().Draw(t, fmt.Sprintf("validDomain%d", i))
			case 1:
				// Domain with v2fly prefix
				lines[i] = genDomainLineWithPrefix().Draw(t, fmt.Sprintf("prefixDomain%d", i))
			case 2:
				// Domain with attributes
				lines[i] = genDomainLineWithAttributes().Draw(t, fmt.Sprintf("attrDomain%d", i))
			case 3:
				// Comment
				lines[i] = genComment().Draw(t, fmt.Sprintf("comment%d", i))
			case 4:
				// Empty line
				lines[i] = genEmptyLine().Draw(t, fmt.Sprintf("empty%d", i))
			case 5:
				// Invalid domain
				lines[i] = genInvalidDomainLine().Draw(t, fmt.Sprintf("invalid%d", i))
			}
		}

		return lines
	})
}

// Property-based tests for DNS routing domain parsing

// TestProperty_ParseDomainLinesNeverReturnsEmptyDomains verifies that parseDomainLines
// never returns empty strings in the domain list
func TestProperty_ParseDomainLinesNeverReturnsEmptyDomains(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lines := genMixedDomainLines().Draw(t, "lines")

		domains, _ := parseDomainLines(lines, "test")

		// Property: No empty domains in result
		for _, domain := range domains {
			if domain == "" {
				t.Fatalf("parseDomainLines returned empty domain")
			}
		}
	})
}

// TestProperty_ParseDomainLinesAllReturnedDomainsAreValid verifies that all domains
// returned by parseDomainLines pass IDNA validation
func TestProperty_ParseDomainLinesAllReturnedDomainsAreValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lines := genMixedDomainLines().Draw(t, "lines")

		domains, _ := parseDomainLines(lines, "test")

		// Property: All returned domains are valid according to validateDomainWithIDNA
		for _, domain := range domains {
			valid, reason := validateDomainWithIDNA(domain)
			if !valid {
				t.Fatalf("parseDomainLines returned invalid domain '%s': %s", domain, reason)
			}
		}
	})
}

// TestProperty_ParseDomainLinesSkipsComments verifies that comment lines
// are never included in the output
func TestProperty_ParseDomainLinesSkipsComments(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate lines with some comments
		numLines := rapid.IntRange(5, 15).Draw(t, "numLines")
		lines := make([]string, numLines)

		for i := range numLines {
			if rapid.Bool().Draw(t, fmt.Sprintf("isComment%d", i)) {
				// Comment line
				lines[i] = genComment().Draw(t, fmt.Sprintf("comment%d", i))
			} else {
				// Valid domain
				lines[i] = genValidDomain().Draw(t, fmt.Sprintf("domain%d", i))
			}
		}

		domains, _ := parseDomainLines(lines, "test")

		// Property: No domain starts with #
		for _, domain := range domains {
			if strings.HasPrefix(domain, "#") {
				t.Fatalf("parseDomainLines returned comment as domain: %s", domain)
			}
		}
	})
}

// TestProperty_ParseDomainLinesStripsV2flyPrefixes verifies that v2fly prefixes
// are correctly removed from domain lines
func TestProperty_ParseDomainLinesStripsV2flyPrefixes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate domain with v2fly prefix
		prefix := genV2flyPrefix().Draw(t, "prefix")
		domain := genValidDomain().Draw(t, "domain")
		line := prefix + domain

		domains, _ := parseDomainLines([]string{line}, "test")

		// Property: Prefix is stripped, domain is preserved
		if len(domains) != 1 {
			t.Fatalf("expected 1 domain, got %d", len(domains))
		}

		if domains[0] != domain {
			t.Fatalf("prefix not stripped correctly: got %s, want %s", domains[0], domain)
		}

		// Property: Result doesn't contain the prefix
		if strings.Contains(domains[0], ":") {
			t.Fatalf("domain still contains colon (prefix not removed): %s", domains[0])
		}
	})
}

// TestProperty_ParseDomainLinesHandlesAttributesCorrectly verifies that attributes
// (like @cn) are stripped from domain lines
func TestProperty_ParseDomainLinesHandlesAttributesCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		domain := genValidDomain().Draw(t, "domain")
		attribute := rapid.StringMatching(`@[a-z]{2}`).Draw(t, "attribute")
		line := domain + " " + attribute

		domains, _ := parseDomainLines([]string{line}, "test")

		// Property: If domain is valid, attribute should be stripped
		// Note: Some generated domains might fail IDNA validation
		if len(domains) == 0 {
			// Domain was invalid, that's okay - just verify it was rejected
			return
		}

		if len(domains) != 1 {
			t.Fatalf("expected 0 or 1 domain, got %d", len(domains))
		}

		if domains[0] != domain {
			t.Fatalf("attribute not stripped correctly: got %s, want %s", domains[0], domain)
		}

		// Property: Result doesn't contain the attribute
		if strings.Contains(domains[0], "@") {
			t.Fatalf("domain still contains @ (attribute not removed): %s", domains[0])
		}
	})
}

// TestProperty_ParseDomainLinesCountsSkippedCorrectly verifies that the skipped
// count is consistent with valid + skipped = total processed
func TestProperty_ParseDomainLinesCountsSkippedCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate lines with known valid and invalid domains
		numValid := rapid.IntRange(1, 10).Draw(t, "numValid")
		numComments := rapid.IntRange(0, 5).Draw(t, "numComments")
		numEmpty := rapid.IntRange(0, 5).Draw(t, "numEmpty")

		lines := make([]string, 0, numValid+numComments+numEmpty)

		// Add valid domains
		for i := range numValid {
			lines = append(lines, genValidDomain().Draw(t, fmt.Sprintf("valid%d", i)))
		}

		// Add some known invalid domains (bare domains without TLD)
		numInvalid := rapid.IntRange(1, 5).Draw(t, "numInvalid")
		for i := range numInvalid {
			// Generate bare domain names (no TLD) - guaranteed to be rejected
			bareName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, fmt.Sprintf("invalid%d", i))
			lines = append(lines, bareName)
		}

		// Add comments (should not be counted as skipped)
		for i := range numComments {
			lines = append(lines, genComment().Draw(t, fmt.Sprintf("comment%d", i)))
		}

		// Add empty lines (should not be counted as skipped)
		for i := range numEmpty {
			lines = append(lines, genEmptyLine().Draw(t, fmt.Sprintf("empty%d", i)))
		}

		domains, skipped := parseDomainLines(lines, "test")

		// Property: valid domains + skipped = total non-empty, non-comment lines
		totalProcessed := len(domains) + skipped
		expectedProcessed := numValid + numInvalid

		if totalProcessed != expectedProcessed {
			t.Fatalf("count mismatch: got %d valid + %d skipped = %d, want %d",
				len(domains), skipped, totalProcessed, expectedProcessed)
		}

		// Property: skipped count should equal numInvalid (all bare domains without TLD)
		if skipped != numInvalid {
			t.Fatalf("skipped count mismatch: got %d, expected %d", skipped, numInvalid)
		}
	})
}

// TestProperty_ValidateDomainWithIDNARejectsBareDomains verifies that domains
// without a TLD (like "youtube", "instagram") are always rejected
func TestProperty_ValidateDomainWithIDNARejectsBareDomains(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a bare domain (single label, no dot)
		bareDomain := rapid.StringMatching(`[a-z]{3,15}`).Draw(t, "bareDomain")

		valid, reason := validateDomainWithIDNA(bareDomain)

		// Property: Bare domains must be rejected
		if valid {
			t.Fatalf("bare domain incorrectly accepted: %s", bareDomain)
		}

		// Property: Reason should mention missing TLD
		if !strings.Contains(reason, "TLD") && !strings.Contains(reason, "dot") {
			t.Fatalf("rejection reason doesn't mention TLD: %s", reason)
		}
	})
}

// TestProperty_ValidateDomainWithIDNAAcceptsValidDomains verifies that
// properly formatted domains are accepted
func TestProperty_ValidateDomainWithIDNAAcceptsValidDomains(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		domain := genValidDomain().Draw(t, "domain")

		valid, reason := validateDomainWithIDNA(domain)

		// Property: Valid domains must be accepted
		if !valid {
			t.Fatalf("valid domain rejected: %s (reason: %s)", domain, reason)
		}
	})
}

// TestProperty_ValidateDomainWithIDNARejectsEmptyString verifies that
// empty strings are always rejected
func TestProperty_ValidateDomainWithIDNARejectsEmptyString(t *testing.T) {
	valid, reason := validateDomainWithIDNA("")

	if valid {
		t.Fatalf("empty string incorrectly accepted as domain")
	}

	if !strings.Contains(reason, "empty") {
		t.Fatalf("rejection reason doesn't mention empty: %s", reason)
	}
}

// TestProperty_ParseDomainLinesIdempotent verifies that parsing the same lines
// multiple times produces the same result
func TestProperty_ParseDomainLinesIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lines := genMixedDomainLines().Draw(t, "lines")

		// Parse twice
		domains1, skipped1 := parseDomainLines(lines, "test")
		domains2, skipped2 := parseDomainLines(lines, "test")

		// Property: Results should be identical
		if len(domains1) != len(domains2) {
			t.Fatalf("domain count differs: %d vs %d", len(domains1), len(domains2))
		}

		if skipped1 != skipped2 {
			t.Fatalf("skipped count differs: %d vs %d", skipped1, skipped2)
		}

		for i := range domains1 {
			if domains1[i] != domains2[i] {
				t.Fatalf("domain at index %d differs: %s vs %s", i, domains1[i], domains2[i])
			}
		}
	})
}

// TestProperty_ParseDomainLinesPreservesOrder verifies that the order of
// valid domains is preserved from the input
func TestProperty_ParseDomainLinesPreservesOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 3-10 unique valid domains
		numDomains := rapid.IntRange(3, 10).Draw(t, "numDomains")
		expectedDomains := make([]string, 0, numDomains)
		seen := make(map[string]bool)

		// Generate unique domains to avoid deduplication affecting the test
		for len(expectedDomains) < numDomains {
			domain := genValidDomain().Draw(t, fmt.Sprintf("domain%d", len(expectedDomains)))
			if !seen[domain] {
				seen[domain] = true
				expectedDomains = append(expectedDomains, domain)
			}
		}

		// Parse the domains
		domains, _ := parseDomainLines(expectedDomains, "test")

		// Property: All domains should be accepted (our generator creates valid domains)
		// and order should be preserved
		if len(domains) != len(expectedDomains) {
			t.Fatalf("domain count mismatch: got %d, want %d (some generated domains may have failed IDNA)",
				len(domains), len(expectedDomains))
		}

		for i := range domains {
			if domains[i] != expectedDomains[i] {
				t.Fatalf("order not preserved at index %d: got %s, want %s",
					i, domains[i], expectedDomains[i])
			}
		}
	})
}
