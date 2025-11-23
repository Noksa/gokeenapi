package gokeenrestapi

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
	"pgregory.net/rapid"
)

// Generator utilities for WireGuard configuration property-based testing

// ASCParameters represents the ASC (Allowed Source Check) parameters
type ASCParameters struct {
	Jc   string
	Jmin string
	Jmax string
	S1   string
	S2   string
	H1   string
	H2   string
	H3   string
	H4   string
}

// genASCParameter generates a valid ASC parameter value (numeric string)
func genASCParameter() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// ASC parameters are typically numeric values
		value := rapid.IntRange(0, 100).Draw(t, "value")
		return fmt.Sprintf("%d", value)
	})
}

// genASCParameters generates a complete set of ASC parameters
func genASCParameters() *rapid.Generator[ASCParameters] {
	return rapid.Custom(func(t *rapid.T) ASCParameters {
		return ASCParameters{
			Jc:   genASCParameter().Draw(t, "Jc"),
			Jmin: genASCParameter().Draw(t, "Jmin"),
			Jmax: genASCParameter().Draw(t, "Jmax"),
			S1:   genASCParameter().Draw(t, "S1"),
			S2:   genASCParameter().Draw(t, "S2"),
			H1:   genASCParameter().Draw(t, "H1"),
			H2:   genASCParameter().Draw(t, "H2"),
			H3:   genASCParameter().Draw(t, "H3"),
			H4:   genASCParameter().Draw(t, "H4"),
		}
	})
}

// genWireGuardConfWithMissingParams generates a .conf file with some ASC parameters missing
func genWireGuardConfWithMissingParams() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate which parameters to include (at least one missing)
		includeJc := rapid.Bool().Draw(t, "includeJc")
		includeJmin := rapid.Bool().Draw(t, "includeJmin")
		includeJmax := rapid.Bool().Draw(t, "includeJmax")
		includeS1 := rapid.Bool().Draw(t, "includeS1")
		includeS2 := rapid.Bool().Draw(t, "includeS2")
		includeH1 := rapid.Bool().Draw(t, "includeH1")
		includeH2 := rapid.Bool().Draw(t, "includeH2")
		includeH3 := rapid.Bool().Draw(t, "includeH3")
		includeH4 := rapid.Bool().Draw(t, "includeH4")

		// Ensure at least one is missing
		allIncluded := includeJc && includeJmin && includeJmax &&
			includeS1 && includeS2 && includeH1 && includeH2 && includeH3 && includeH4

		if allIncluded {
			// Force at least one to be missing
			switch rapid.IntRange(0, 8).Draw(t, "missingParam") {
			case 0:
				includeJc = false
			case 1:
				includeJmin = false
			case 2:
				includeJmax = false
			case 3:
				includeS1 = false
			case 4:
				includeS2 = false
			case 5:
				includeH1 = false
			case 6:
				includeH2 = false
			case 7:
				includeH3 = false
			case 8:
				includeH4 = false
			}
		}

		params := genASCParameters().Draw(t, "params")

		// Build config with only included parameters
		var lines []string
		lines = append(lines, "[Interface]")
		lines = append(lines, "PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=")
		lines = append(lines, "Address = 10.0.0.2/24")
		lines = append(lines, "DNS = 8.8.8.8")

		if includeJc {
			lines = append(lines, fmt.Sprintf("Jc = %s", params.Jc))
		}
		if includeJmin {
			lines = append(lines, fmt.Sprintf("Jmin = %s", params.Jmin))
		}
		if includeJmax {
			lines = append(lines, fmt.Sprintf("Jmax = %s", params.Jmax))
		}
		if includeS1 {
			lines = append(lines, fmt.Sprintf("S1 = %s", params.S1))
		}
		if includeS2 {
			lines = append(lines, fmt.Sprintf("S2 = %s", params.S2))
		}
		if includeH1 {
			lines = append(lines, fmt.Sprintf("H1 = %s", params.H1))
		}
		if includeH2 {
			lines = append(lines, fmt.Sprintf("H2 = %s", params.H2))
		}
		if includeH3 {
			lines = append(lines, fmt.Sprintf("H3 = %s", params.H3))
		}
		if includeH4 {
			lines = append(lines, fmt.Sprintf("H4 = %s", params.H4))
		}

		lines = append(lines, "")
		lines = append(lines, "[Peer]")
		lines = append(lines, "PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=")
		lines = append(lines, "Endpoint = example.com:51820")
		lines = append(lines, "AllowedIPs = 0.0.0.0/0")
		lines = append(lines, "PersistentKeepalive = 25")

		return strings.Join(lines, "\n")
	})
}

// Helper functions for ASC parameter extraction and validation

// extractASCParametersFromConf extracts ASC parameters from a .conf file path
func extractASCParametersFromConf(confPath string) (ASCParameters, error) {
	cfg, err := ini.Load(confPath)
	if err != nil {
		return ASCParameters{}, err
	}

	interfaceSection, err := cfg.GetSection("Interface")
	if err != nil {
		return ASCParameters{}, err
	}

	params := ASCParameters{}

	// Try to extract each parameter
	if key, err := interfaceSection.GetKey("Jc"); err == nil {
		params.Jc = key.String()
	} else {
		return params, fmt.Errorf("missing Jc parameter")
	}

	if key, err := interfaceSection.GetKey("Jmin"); err == nil {
		params.Jmin = key.String()
	} else {
		return params, fmt.Errorf("missing Jmin parameter")
	}

	if key, err := interfaceSection.GetKey("Jmax"); err == nil {
		params.Jmax = key.String()
	} else {
		return params, fmt.Errorf("missing Jmax parameter")
	}

	if key, err := interfaceSection.GetKey("S1"); err == nil {
		params.S1 = key.String()
	} else {
		return params, fmt.Errorf("missing S1 parameter")
	}

	if key, err := interfaceSection.GetKey("S2"); err == nil {
		params.S2 = key.String()
	} else {
		return params, fmt.Errorf("missing S2 parameter")
	}

	if key, err := interfaceSection.GetKey("H1"); err == nil {
		params.H1 = key.String()
	} else {
		return params, fmt.Errorf("missing H1 parameter")
	}

	if key, err := interfaceSection.GetKey("H2"); err == nil {
		params.H2 = key.String()
	} else {
		return params, fmt.Errorf("missing H2 parameter")
	}

	if key, err := interfaceSection.GetKey("H3"); err == nil {
		params.H3 = key.String()
	} else {
		return params, fmt.Errorf("missing H3 parameter")
	}

	if key, err := interfaceSection.GetKey("H4"); err == nil {
		params.H4 = key.String()
	} else {
		return params, fmt.Errorf("missing H4 parameter")
	}

	return params, nil
}

// ascParametersEqual checks if two ASCParameters are equal
func ascParametersEqual(p1, p2 ASCParameters) bool {
	return p1.Jc == p2.Jc &&
		p1.Jmin == p2.Jmin &&
		p1.Jmax == p2.Jmax &&
		p1.S1 == p2.S1 &&
		p1.S2 == p2.S2 &&
		p1.H1 == p2.H1 &&
		p1.H2 == p2.H2 &&
		p1.H3 == p2.H3 &&
		p1.H4 == p2.H4
}

// ascParametersDiff returns the parameters that differ between two ASCParameters
func ascParametersDiff(p1, p2 ASCParameters) []string {
	var diffs []string

	if p1.Jc != p2.Jc {
		diffs = append(diffs, "Jc")
	}
	if p1.Jmin != p2.Jmin {
		diffs = append(diffs, "Jmin")
	}
	if p1.Jmax != p2.Jmax {
		diffs = append(diffs, "Jmax")
	}
	if p1.S1 != p2.S1 {
		diffs = append(diffs, "S1")
	}
	if p1.S2 != p2.S2 {
		diffs = append(diffs, "S2")
	}
	if p1.H1 != p2.H1 {
		diffs = append(diffs, "H1")
	}
	if p1.H2 != p2.H2 {
		diffs = append(diffs, "H2")
	}
	if p1.H3 != p2.H3 {
		diffs = append(diffs, "H3")
	}
	if p1.H4 != p2.H4 {
		diffs = append(diffs, "H4")
	}

	return diffs
}

// writeConfToTempFile writes config content to a temporary file and returns the path
func writeConfToTempFile(t *rapid.T, content string) string {
	tmpDir, err := os.MkdirTemp("", "awgconf-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	confPath := filepath.Join(tmpDir, "test.conf")

	err = os.WriteFile(confPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	return confPath
}

// Property-based tests for WireGuard configuration

// Feature: property-based-testing, Property 13: ASC parameter extraction completeness
// Validates: Requirements 4.1
func TestWireGuardASCParametersExtractedFromValidConfig(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a complete WireGuard config with all ASC parameters
		expectedParams := genASCParameters().Draw(t, "expectedParams")

		// Create config content with these parameters
		confContent := fmt.Sprintf(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
DNS = 8.8.8.8
Jc = %s
Jmin = %s
Jmax = %s
S1 = %s
S2 = %s
H1 = %s
H2 = %s
H3 = %s
H4 = %s

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`,
			expectedParams.Jc, expectedParams.Jmin, expectedParams.Jmax,
			expectedParams.S1, expectedParams.S2,
			expectedParams.H1, expectedParams.H2, expectedParams.H3, expectedParams.H4)

		// Write to temp file
		confPath := writeConfToTempFile(t, confContent)

		// Extract parameters
		extractedParams, err := extractASCParametersFromConf(confPath)

		// Property: All ASC parameters should be extracted correctly
		if err != nil {
			t.Fatalf("Failed to extract ASC parameters: %v", err)
		}

		if !ascParametersEqual(expectedParams, extractedParams) {
			t.Fatalf("Extracted parameters don't match expected.\nExpected: %+v\nGot: %+v",
				expectedParams, extractedParams)
		}
	})
}

// Feature: property-based-testing, Property 14: INI formatting variations are normalized
// Validates: Requirements 4.2
func TestWireGuardINIWhitespaceVariationsProduceSameParameters(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate ASC parameters
		expectedParams := genASCParameters().Draw(t, "expectedParams")

		// Generate two configs with the same parameters but different whitespace
		ws1_1 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws1_1")
		ws2_1 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws2_1")

		ws1_2 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws1_2")
		ws2_2 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws2_2")

		// Create two configs with different formatting
		confContent1 := fmt.Sprintf(`[Interface]
PrivateKey%s=%scOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address%s=%s10.0.0.2/24
Jc%s=%s%s
Jmin%s=%s%s
Jmax%s=%s%s
S1%s=%s%s
S2%s=%s%s
H1%s=%s%s
H2%s=%s%s
H3%s=%s%s
H4%s=%s%s

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=`,
			ws1_1, ws2_1,
			ws1_1, ws2_1,
			ws1_1, ws2_1, expectedParams.Jc,
			ws1_1, ws2_1, expectedParams.Jmin,
			ws1_1, ws2_1, expectedParams.Jmax,
			ws1_1, ws2_1, expectedParams.S1,
			ws1_1, ws2_1, expectedParams.S2,
			ws1_1, ws2_1, expectedParams.H1,
			ws1_1, ws2_1, expectedParams.H2,
			ws1_1, ws2_1, expectedParams.H3,
			ws1_1, ws2_1, expectedParams.H4)

		confContent2 := fmt.Sprintf(`[Interface]
PrivateKey%s=%scOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address%s=%s10.0.0.2/24
Jc%s=%s%s
Jmin%s=%s%s
Jmax%s=%s%s
S1%s=%s%s
S2%s=%s%s
H1%s=%s%s
H2%s=%s%s
H3%s=%s%s
H4%s=%s%s

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=`,
			ws1_2, ws2_2,
			ws1_2, ws2_2,
			ws1_2, ws2_2, expectedParams.Jc,
			ws1_2, ws2_2, expectedParams.Jmin,
			ws1_2, ws2_2, expectedParams.Jmax,
			ws1_2, ws2_2, expectedParams.S1,
			ws1_2, ws2_2, expectedParams.S2,
			ws1_2, ws2_2, expectedParams.H1,
			ws1_2, ws2_2, expectedParams.H2,
			ws1_2, ws2_2, expectedParams.H3,
			ws1_2, ws2_2, expectedParams.H4)

		// Write to temp files
		confPath1 := writeConfToTempFile(t, confContent1)
		confPath2 := writeConfToTempFile(t, confContent2)

		// Extract parameters from both
		params1, err1 := extractASCParametersFromConf(confPath1)
		params2, err2 := extractASCParametersFromConf(confPath2)

		// Property: Both should parse successfully and produce the same values
		if err1 != nil {
			t.Fatalf("Failed to parse config 1: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("Failed to parse config 2: %v", err2)
		}

		if !ascParametersEqual(params1, params2) {
			t.Fatalf("Formatting variations produced different parameters.\nParams1: %+v\nParams2: %+v",
				params1, params2)
		}

		// Also verify they match the expected values
		if !ascParametersEqual(params1, expectedParams) {
			t.Fatalf("Extracted parameters don't match expected.\nExpected: %+v\nGot: %+v",
				expectedParams, params1)
		}
	})
}

// Feature: property-based-testing, Property 15: Missing ASC parameters are reported
// Validates: Requirements 4.3
func TestWireGuardConfigWithMissingASCParametersReturnsError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a config with at least one missing parameter
		confContent := genWireGuardConfWithMissingParams().Draw(t, "confContent")

		// Write to temp file
		confPath := writeConfToTempFile(t, confContent)

		// Try to extract parameters
		_, err := extractASCParametersFromConf(confPath)

		// Property: Extraction should fail with an error indicating missing parameter
		if err == nil {
			t.Fatalf("Expected error for missing parameters, but extraction succeeded")
		}

		// Verify the error message mentions "missing"
		if !strings.Contains(err.Error(), "missing") {
			t.Fatalf("Error should mention 'missing' parameter, got: %v", err)
		}
	})
}

// Feature: property-based-testing, Property 16: ASC parameter diff is accurate
// Validates: Requirements 4.4
func TestWireGuardASCParameterDiffIdentifiesExactDifferences(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate two sets of ASC parameters
		params1 := genASCParameters().Draw(t, "params1")
		params2 := genASCParameters().Draw(t, "params2")

		// Calculate diff
		diffs := ascParametersDiff(params1, params2)

		// Property: Diff should identify exactly those parameters where values differ
		// Check each parameter individually
		if params1.Jc != params2.Jc {
			if !contains(diffs, "Jc") {
				t.Fatalf("Diff should include 'Jc' (values: %s vs %s), but got: %v",
					params1.Jc, params2.Jc, diffs)
			}
		} else {
			if contains(diffs, "Jc") {
				t.Fatalf("Diff should not include 'Jc' (both are %s), but got: %v",
					params1.Jc, diffs)
			}
		}

		if params1.Jmin != params2.Jmin {
			if !contains(diffs, "Jmin") {
				t.Fatalf("Diff should include 'Jmin' (values: %s vs %s), but got: %v",
					params1.Jmin, params2.Jmin, diffs)
			}
		} else {
			if contains(diffs, "Jmin") {
				t.Fatalf("Diff should not include 'Jmin' (both are %s), but got: %v",
					params1.Jmin, diffs)
			}
		}

		if params1.Jmax != params2.Jmax {
			if !contains(diffs, "Jmax") {
				t.Fatalf("Diff should include 'Jmax' (values: %s vs %s), but got: %v",
					params1.Jmax, params2.Jmax, diffs)
			}
		} else {
			if contains(diffs, "Jmax") {
				t.Fatalf("Diff should not include 'Jmax' (both are %s), but got: %v",
					params1.Jmax, diffs)
			}
		}

		if params1.S1 != params2.S1 {
			if !contains(diffs, "S1") {
				t.Fatalf("Diff should include 'S1' (values: %s vs %s), but got: %v",
					params1.S1, params2.S1, diffs)
			}
		} else {
			if contains(diffs, "S1") {
				t.Fatalf("Diff should not include 'S1' (both are %s), but got: %v",
					params1.S1, diffs)
			}
		}

		if params1.S2 != params2.S2 {
			if !contains(diffs, "S2") {
				t.Fatalf("Diff should include 'S2' (values: %s vs %s), but got: %v",
					params1.S2, params2.S2, diffs)
			}
		} else {
			if contains(diffs, "S2") {
				t.Fatalf("Diff should not include 'S2' (both are %s), but got: %v",
					params1.S2, diffs)
			}
		}

		if params1.H1 != params2.H1 {
			if !contains(diffs, "H1") {
				t.Fatalf("Diff should include 'H1' (values: %s vs %s), but got: %v",
					params1.H1, params2.H1, diffs)
			}
		} else {
			if contains(diffs, "H1") {
				t.Fatalf("Diff should not include 'H1' (both are %s), but got: %v",
					params1.H1, diffs)
			}
		}

		if params1.H2 != params2.H2 {
			if !contains(diffs, "H2") {
				t.Fatalf("Diff should include 'H2' (values: %s vs %s), but got: %v",
					params1.H2, params2.H2, diffs)
			}
		} else {
			if contains(diffs, "H2") {
				t.Fatalf("Diff should not include 'H2' (both are %s), but got: %v",
					params1.H2, diffs)
			}
		}

		if params1.H3 != params2.H3 {
			if !contains(diffs, "H3") {
				t.Fatalf("Diff should include 'H3' (values: %s vs %s), but got: %v",
					params1.H3, params2.H3, diffs)
			}
		} else {
			if contains(diffs, "H3") {
				t.Fatalf("Diff should not include 'H3' (both are %s), but got: %v",
					params1.H3, diffs)
			}
		}

		if params1.H4 != params2.H4 {
			if !contains(diffs, "H4") {
				t.Fatalf("Diff should include 'H4' (values: %s vs %s), but got: %v",
					params1.H4, params2.H4, diffs)
			}
		} else {
			if contains(diffs, "H4") {
				t.Fatalf("Diff should not include 'H4' (both are %s), but got: %v",
					params1.H4, diffs)
			}
		}
	})
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// isValidASCParameter checks if a parameter value is valid (numeric string)
func isValidASCParameter(value string) bool {
	// ASC parameters should be numeric strings
	if value == "" {
		return false
	}

	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// Feature: property-based-testing, Property 17: ASC parameter validation
// Validates: Requirements 4.5
func TestWireGuardASCParametersValidateAsNumericStrings(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate ASC parameters (which should all be valid)
		params := genASCParameters().Draw(t, "params")

		// Property: All generated parameters should be valid
		if !isValidASCParameter(params.Jc) {
			t.Fatalf("Generated Jc parameter is invalid: %s", params.Jc)
		}
		if !isValidASCParameter(params.Jmin) {
			t.Fatalf("Generated Jmin parameter is invalid: %s", params.Jmin)
		}
		if !isValidASCParameter(params.Jmax) {
			t.Fatalf("Generated Jmax parameter is invalid: %s", params.Jmax)
		}
		if !isValidASCParameter(params.S1) {
			t.Fatalf("Generated S1 parameter is invalid: %s", params.S1)
		}
		if !isValidASCParameter(params.S2) {
			t.Fatalf("Generated S2 parameter is invalid: %s", params.S2)
		}
		if !isValidASCParameter(params.H1) {
			t.Fatalf("Generated H1 parameter is invalid: %s", params.H1)
		}
		if !isValidASCParameter(params.H2) {
			t.Fatalf("Generated H2 parameter is invalid: %s", params.H2)
		}
		if !isValidASCParameter(params.H3) {
			t.Fatalf("Generated H3 parameter is invalid: %s", params.H3)
		}
		if !isValidASCParameter(params.H4) {
			t.Fatalf("Generated H4 parameter is invalid: %s", params.H4)
		}

		// Also test that invalid values are rejected
		invalidValues := []string{"", "abc", "12.5", "-10", "1a2", " 10", "10 "}
		for _, invalid := range invalidValues {
			if isValidASCParameter(invalid) {
				t.Fatalf("Validation should reject invalid value: %s", invalid)
			}
		}
	})
}
