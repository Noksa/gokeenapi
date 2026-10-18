package gokeenrestapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/pmezard/go-difflib/difflib"
	"gopkg.in/ini.v1"
)

type awgRange struct{ Start, End int }

type awg3Params struct {
	Present                bool
	HeaderProtectionKey    string
	ContentPaddingAddition awgRange
	RekeyAfterTime         awgRange
	RekeyTimeout           awgRange
	RejectAfterTime        awgRange
	KeepaliveTimeout       awgRange
	MaxHandshakeAttempts   awgRange
	RandomTrailers         bool
	DisableCookies         bool
}

func confHasAWG3Fields(contents []byte) bool {
	cfg, err := ini.Load(contents)
	if err != nil {
		return false
	}
	section, err := cfg.GetSection("Interface")
	if err != nil {
		return false
	}
	for _, name := range []string{
		"HeaderProtectionKey", "ContentPaddingAddition", "RekeyAfterTime",
		"RekeyTimeout", "RejectAfterTime", "KeepaliveTimeout",
		"MaxHandshakeAttempts", "RandomTrailers", "DisableCookies",
	} {
		if _, err := section.GetKey(name); err == nil {
			return true
		}
	}
	return false
}

func (a ascParams) hasAWG3() bool { return a.AWG3.Present }

func parseAWGRange(name, value string) (awgRange, error) {
	return parseBoundedRange(name, value, 65535)
}

func parseBoundedRange(name, value string, maximum int) (awgRange, error) {
	parts := strings.Split(value, "-")
	if len(parts) < 1 || len(parts) > 2 {
		return awgRange{}, fmt.Errorf("invalid %s range %q", name, value)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start < 0 || start > maximum {
		return awgRange{}, fmt.Errorf("invalid %s range %q", name, value)
	}
	end := start
	if len(parts) == 2 {
		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || end < start || end > maximum {
			return awgRange{}, fmt.Errorf("invalid %s range %q", name, value)
		}
	}
	return awgRange{start, end}, nil
}

func (a *ascParams) parseAWG3(section *ini.Section) error {
	fields := []struct {
		name string
		dst  *awgRange
	}{
		{"ContentPaddingAddition", &a.AWG3.ContentPaddingAddition},
		{"RekeyAfterTime", &a.AWG3.RekeyAfterTime},
		{"RekeyTimeout", &a.AWG3.RekeyTimeout},
		{"RejectAfterTime", &a.AWG3.RejectAfterTime},
		{"KeepaliveTimeout", &a.AWG3.KeepaliveTimeout},
		{"MaxHandshakeAttempts", &a.AWG3.MaxHandshakeAttempts},
	}
	for _, field := range fields {
		key, err := section.GetKey(field.name)
		if err != nil {
			continue
		}
		a.AWG3.Present = true
		*field.dst, err = parseAWGRange(field.name, key.String())
		if err != nil {
			return err
		}
	}
	for _, field := range []struct {
		name string
		dst  *bool
	}{
		{"RandomTrailers", &a.AWG3.RandomTrailers},
		{"DisableCookies", &a.AWG3.DisableCookies},
	} {
		key, err := section.GetKey(field.name)
		if err != nil {
			continue
		}
		a.AWG3.Present = true
		switch strings.ToLower(strings.TrimSpace(key.String())) {
		case "on", "true", "1":
			*field.dst = true
		case "off", "false", "0":
			*field.dst = false
		default:
			return fmt.Errorf("invalid %s value %q", field.name, key.String())
		}
	}
	if key, err := section.GetKey("HeaderProtectionKey"); err == nil {
		a.AWG3.Present = true
		a.AWG3.HeaderProtectionKey = key.String()
		decoded, err := base64.StdEncoding.DecodeString(a.AWG3.HeaderProtectionKey)
		if err != nil || len(decoded) != 32 {
			return fmt.Errorf("HeaderProtectionKey must be a base64-encoded 32-byte key")
		}
	}
	return nil
}

func (a ascParams) awg3Payload() (map[string]any, error) {
	values := map[string]any{}
	for _, field := range []struct{ name, value string }{
		{"jc", a.Jc}, {"jmin", a.Jmin}, {"jmax", a.Jmax},
		{"s1", a.S1}, {"s2", a.S2}, {"s3", a.S3}, {"s4", a.S4},
	} {
		v := defaultZero(field.value)
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid %s value %q", field.name, v)
		}
		values[field.name] = n
	}
	if a.AWG3.HeaderProtectionKey != "" {
		for _, name := range []string{"s1", "s2", "s3", "s4"} {
			if values[name].(int) < 12 {
				return nil, fmt.Errorf("%s must be at least 12 when HeaderProtectionKey is set", strings.ToUpper(name))
			}
		}
	}
	for _, field := range []struct{ name, value string }{
		{"h1", a.H1}, {"h2", a.H2}, {"h3", a.H3}, {"h4", a.H4},
	} {
		v := defaultZero(field.value)
		if _, err := parseBoundedRange(field.name, v, 4294967295); err != nil {
			return nil, err
		}
		values[field.name] = v
	}
	for _, field := range []struct{ name, value string }{
		{"i1", a.I1}, {"i2", a.I2}, {"i3", a.I3}, {"i4", a.I4}, {"i5", a.I5},
	} {
		values[field.name] = field.value
	}
	values["header-protection-key"] = a.AWG3.HeaderProtectionKey
	for _, field := range []struct {
		name  string
		value awgRange
	}{
		{"content-padding-addition", a.AWG3.ContentPaddingAddition},
		{"rekey-after-time", a.AWG3.RekeyAfterTime},
		{"rekey-timeout", a.AWG3.RekeyTimeout},
		{"reject-after-time", a.AWG3.RejectAfterTime},
		{"keepalive-timeout", a.AWG3.KeepaliveTimeout},
		{"max-handshake-attempts", a.AWG3.MaxHandshakeAttempts},
	} {
		values[field.name+"-start"] = field.value.Start
		values[field.name+"-end"] = field.value.End
	}
	values["random-trailers"] = boolInt(a.AWG3.RandomTrailers)
	values["disable-cookies"] = boolInt(a.AWG3.DisableCookies)
	return values, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func requireAWG3Firmware() error {
	info, err := Common.Version()
	if err != nil {
		return err
	}
	numeric := versionNumericRe.FindString(info.Title)
	current, err := version.NewVersion(numeric)
	if err != nil {
		return fmt.Errorf("cannot determine KeeneticOS version %q for AWG 3.1", info.Title)
	}
	minimum, _ := version.NewVersion("5.2")
	if current.LessThan(minimum) {
		return fmt.Errorf("AWG 3.1 requires KeeneticOS 5.2+; router runs %s", info.Title)
	}
	if matches := regexp.MustCompile(`^5\.0?2\.A\.(\d+)`).FindStringSubmatch(info.Title); len(matches) == 2 {
		build, _ := strconv.Atoi(matches[1])
		if build < 11 {
			return fmt.Errorf("AWG 3.1 requires KeeneticOS 5.02.A.11+; router runs %s", info.Title)
		}
	}
	return nil
}

type awg3Plan struct {
	asc     map[string]any
	current map[string]any
	peer    []gokeenrestapimodels.ParseRequest
}

func planAWG3(interfaceID string, asc ascParams, peer peerParams) (awg3Plan, error) {
	if err := Checks.CheckInterfaceId(interfaceID); err != nil {
		return awg3Plan{}, err
	}
	if err := Checks.CheckInterfaceExists(interfaceID); err != nil {
		return awg3Plan{}, err
	}
	if err := requireAWG3Firmware(); err != nil {
		return awg3Plan{}, err
	}
	ascPayload, err := asc.awg3Payload()
	if err != nil {
		return awg3Plan{}, err
	}
	current, err := getAWG3Current(interfaceID)
	if err != nil {
		return awg3Plan{}, err
	}
	interfaceDetails, err := Interface.GetInterfaceViaRciShowScInterfaces(interfaceID)
	if err != nil {
		return awg3Plan{}, err
	}
	commands := buildPeerCommands(interfaceID, peer, interfaceDetails.Wireguard.Peer)
	if peer.PublicKey == "" {
		commands = nil
	}
	return awg3Plan{asc: ascPayload, current: current, peer: commands}, nil
}

func getAWG3Current(interfaceID string) (map[string]any, error) {
	currentBody, err := Common.ExecuteGetSubPath("/rci/show/rc/interface/" + interfaceID + "/wireguard/asc")
	if err != nil {
		return nil, err
	}
	var current map[string]any
	if err := json.Unmarshal(currentBody, &current); err != nil {
		return nil, fmt.Errorf("invalid AWG 3.1 router state: %w", err)
	}
	if current == nil {
		return nil, fmt.Errorf("empty AWG 3.1 router state")
	}
	return current, nil
}

func (p awg3Plan) ascChanged() bool {
	for name, want := range p.asc {
		got, ok := p.current[name]
		if !ok || fmt.Sprint(got) != fmt.Sprint(want) {
			return true
		}
	}
	return false
}

func configureAWG3(interfaceID string, asc ascParams, peer peerParams) error {
	plan, err := planAWG3(interfaceID, asc, peer)
	if err != nil {
		return err
	}
	if !plan.ascChanged() && len(plan.peer) == 0 {
		gokeenlog.InfoSubStepf("Interface %s is already up to date", interfaceID)
		return nil
	}
	return gokeenspinner.WrapWithSpinner("Updating "+interfaceID+" AWG 3.1 configuration", func() error {
		if plan.ascChanged() {
			body := map[string]any{"interface": map[string]any{interfaceID: map[string]any{"wireguard": map[string]any{"asc": plan.asc}}}}
			client, err := Common.GetApiClient()
			if err != nil {
				return err
			}
			response, err := client.R().SetBody(body).Post("/rci/")
			if err != nil {
				return err
			}
			if response == nil || response.StatusCode() != http.StatusOK {
				return fmt.Errorf("AWG 3.1 RCI returned an unsuccessful HTTP response")
			}
			if err := checkAWG3Response(response.Body()); err != nil {
				return err
			}
			actual, err := getAWG3Current(interfaceID)
			if err != nil {
				return err
			}
			if (awg3Plan{asc: plan.asc, current: actual}).ascChanged() {
				return fmt.Errorf("router did not apply all AWG 3.1 parameters")
			}
		}
		if len(plan.peer) > 0 {
			if _, err := Common.ExecutePostParse(plan.peer...); err != nil {
				return err
			}
		}
		_, err := Common.ExecutePostParse(gokeenrestapimodels.ParseRequest{Parse: "system configuration save"})
		return err
	})
}

func checkAWG3Response(body []byte) error {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("invalid AWG 3.1 RCI response: %w", err)
	}
	var walk func(any) error
	walk = func(node any) error {
		switch v := node.(type) {
		case map[string]any:
			if status, ok := v["status"].(string); ok {
				if strings.EqualFold(status, "error") {
					return fmt.Errorf("AWG 3.1 RCI rejected configuration: %v", v["message"])
				}
			}
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return err
	}
	if root == nil {
		return fmt.Errorf("AWG 3.1 RCI response is empty")
	}
	return nil
}

func diffAWG3(confPath, interfaceID string, asc ascParams, peer peerParams) (string, error) {
	plan, err := planAWG3(interfaceID, asc, peer)
	if err != nil {
		return "", err
	}
	if !plan.ascChanged() && len(plan.peer) == 0 {
		return "", nil
	}
	current, err := Interface.GetInterfaceViaRciShowScInterfaces(interfaceID)
	if err != nil {
		return "", err
	}
	oldText := renderCanonicalConf(current) + renderAWG3Summary(plan.current)
	newText := renderCanonicalConfFromParams(current, asc, peer, extractMtuFromConf(confPath)) + renderAWG3Summary(plan.asc)
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(oldText), B: difflib.SplitLines(newText),
		FromFile: interfaceID + " (current)", ToFile: filepath.Base(confPath) + " (new)", Context: 3,
	})
}

func renderAWG3Summary(values map[string]any) string {
	var b strings.Builder
	for _, item := range []struct{ label, key string }{
		{"HeaderProtectionKey", "header-protection-key"},
		{"ContentPaddingAddition", "content-padding-addition"},
		{"RekeyAfterTime", "rekey-after-time"},
		{"RekeyTimeout", "rekey-timeout"},
		{"RejectAfterTime", "reject-after-time"},
		{"KeepaliveTimeout", "keepalive-timeout"},
		{"MaxHandshakeAttempts", "max-handshake-attempts"},
		{"RandomTrailers", "random-trailers"},
		{"DisableCookies", "disable-cookies"},
	} {
		if item.key == "header-protection-key" {
			if key, ok := values[item.key].(string); ok && key != "" {
				digest := sha256.Sum256([]byte(key))
				fmt.Fprintf(&b, "%s = sha256:%x\n", item.label, digest[:4])
			}
			continue
		}
		if item.key == "random-trailers" || item.key == "disable-cookies" {
			if value, ok := values[item.key]; ok {
				fmt.Fprintf(&b, "%s = %v\n", item.label, value)
			}
			continue
		}
		start, startOK := values[item.key+"-start"]
		end, endOK := values[item.key+"-end"]
		if startOK && endOK {
			fmt.Fprintf(&b, "%s = %v-%v\n", item.label, start, end)
		}
	}
	return b.String()
}
