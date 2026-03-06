package gokeenrestapi

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		includeJc := rapid.Bool().Draw(t, "includeJc")
		includeJmin := rapid.Bool().Draw(t, "includeJmin")
		includeJmax := rapid.Bool().Draw(t, "includeJmax")
		includeS1 := rapid.Bool().Draw(t, "includeS1")
		includeS2 := rapid.Bool().Draw(t, "includeS2")
		includeH1 := rapid.Bool().Draw(t, "includeH1")
		includeH2 := rapid.Bool().Draw(t, "includeH2")
		includeH3 := rapid.Bool().Draw(t, "includeH3")
		includeH4 := rapid.Bool().Draw(t, "includeH4")

		allIncluded := includeJc && includeJmin && includeJmax &&
			includeS1 && includeS2 && includeH1 && includeH2 && includeH3 && includeH4

		if allIncluded {
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
func writeConfToTempFile(content string) string {
	tmpDir, err := os.MkdirTemp("", "awgconf-test-*")
	Expect(err).NotTo(HaveOccurred(), "Failed to create temp dir")

	confPath := filepath.Join(tmpDir, "test.conf")
	err = os.WriteFile(confPath, []byte(content), 0644)
	Expect(err).NotTo(HaveOccurred(), "Failed to write temp config file")

	return confPath
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// isValidASCParameter checks if a parameter value is valid (numeric string)
func isValidASCParameter(value string) bool {
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

var _ = Describe("Property: ASC Parameter Extraction", func() {
	It("should extract all ASC parameters from a valid config", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			expectedParams := genASCParameters().Draw(t, "expectedParams")

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

			confPath := writeConfToTempFile(confContent)

			extractedParams, err := extractASCParametersFromConf(confPath)
			Expect(err).NotTo(HaveOccurred(), "Failed to extract ASC parameters")
			Expect(ascParametersEqual(expectedParams, extractedParams)).To(BeTrue(),
				"Extracted parameters don't match expected.\nExpected: %+v\nGot: %+v",
				expectedParams, extractedParams)
		})
	})

	It("should normalize INI whitespace variations", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			expectedParams := genASCParameters().Draw(t, "expectedParams")

			ws1_1 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws1_1")
			ws2_1 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws2_1")
			ws1_2 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws1_2")
			ws2_2 := rapid.SampledFrom([]string{"", " ", "  ", "\t"}).Draw(t, "ws2_2")

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

			confPath1 := writeConfToTempFile(confContent1)
			confPath2 := writeConfToTempFile(confContent2)

			params1, err1 := extractASCParametersFromConf(confPath1)
			params2, err2 := extractASCParametersFromConf(confPath2)

			Expect(err1).NotTo(HaveOccurred(), "Failed to parse config 1")
			Expect(err2).NotTo(HaveOccurred(), "Failed to parse config 2")
			Expect(ascParametersEqual(params1, params2)).To(BeTrue(),
				"Formatting variations produced different parameters.\nParams1: %+v\nParams2: %+v",
				params1, params2)
			Expect(ascParametersEqual(params1, expectedParams)).To(BeTrue(),
				"Extracted parameters don't match expected.\nExpected: %+v\nGot: %+v",
				expectedParams, params1)
		})
	})

	It("should report error for missing ASC parameters", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			confContent := genWireGuardConfWithMissingParams().Draw(t, "confContent")
			confPath := writeConfToTempFile(confContent)

			_, err := extractASCParametersFromConf(confPath)
			Expect(err).To(HaveOccurred(), "Expected error for missing parameters")
			Expect(err.Error()).To(ContainSubstring("missing"),
				"Error should mention 'missing' parameter, got: %v", err)
		})
	})
})

var _ = Describe("Property: ASC Parameter Diff", func() {
	It("should accurately identify differing parameters", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			params1 := genASCParameters().Draw(t, "params1")
			params2 := genASCParameters().Draw(t, "params2")

			diffs := ascParametersDiff(params1, params2)

			paramChecks := []struct {
				name string
				v1   string
				v2   string
			}{
				{"Jc", params1.Jc, params2.Jc},
				{"Jmin", params1.Jmin, params2.Jmin},
				{"Jmax", params1.Jmax, params2.Jmax},
				{"S1", params1.S1, params2.S1},
				{"S2", params1.S2, params2.S2},
				{"H1", params1.H1, params2.H1},
				{"H2", params1.H2, params2.H2},
				{"H3", params1.H3, params2.H3},
				{"H4", params1.H4, params2.H4},
			}

			for _, pc := range paramChecks {
				if pc.v1 != pc.v2 {
					Expect(contains(diffs, pc.name)).To(BeTrue(),
						"Diff should include '%s' (values: %s vs %s), but got: %v",
						pc.name, pc.v1, pc.v2, diffs)
				} else {
					Expect(contains(diffs, pc.name)).To(BeFalse(),
						"Diff should not include '%s' (both are %s), but got: %v",
						pc.name, pc.v1, diffs)
				}
			}
		})
	})
})

var _ = Describe("Property: ASC Parameter Validation", func() {
	It("should validate generated parameters as numeric", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			params := genASCParameters().Draw(t, "params")

			allParams := []struct {
				name  string
				value string
			}{
				{"Jc", params.Jc}, {"Jmin", params.Jmin}, {"Jmax", params.Jmax},
				{"S1", params.S1}, {"S2", params.S2},
				{"H1", params.H1}, {"H2", params.H2}, {"H3", params.H3}, {"H4", params.H4},
			}

			for _, p := range allParams {
				Expect(isValidASCParameter(p.value)).To(BeTrue(),
					"Generated %s parameter is invalid: %s", p.name, p.value)
			}

			invalidValues := []string{"", "abc", "12.5", "-10", "1a2", " 10", "10 "}
			for _, invalid := range invalidValues {
				Expect(isValidASCParameter(invalid)).To(BeFalse(),
					"Validation should reject invalid value: %s", invalid)
			}
		})
	})
})
