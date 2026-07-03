package gokeenrestapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/pmezard/go-difflib/difflib"
	"go.uber.org/multierr"
	"gopkg.in/ini.v1"
)

var (
	// AwgConf provides WireGuard (AWG) configuration management functionality
	AwgConf keeneticAwgconf
)

type keeneticAwgconf struct{}

const minAWG2Version = "5.1"

// warnIfAWG2Unsupported logs a warning if the router firmware is below 5.1
// and AWG 2.0 parameters are present in the conf file.
func warnIfAWG2Unsupported(asc ascParams) {
	if !asc.hasAWG2() {
		return
	}

	runtime := gokeencache.GetRuntimeConfig()
	routerVersion := runtime.RouterInfo.Version.Title
	if routerVersion == "" {
		return
	}

	numericVersion := versionNumericRe.FindString(routerVersion)
	if numericVersion == "" {
		return
	}

	currentVer, err := version.NewVersion(numericVersion)
	if err != nil {
		return
	}

	minVer, err := version.NewVersion(minAWG2Version)
	if err != nil {
		return
	}

	if currentVer.LessThan(minVer) {
		gokeenlog.Infof("⚠️  %s: AWG 2.0 parameters (S3, S4, I1-I5) require KeeneticOS %s+. Current firmware: %s. These parameters may be ignored by the router.",
			color.YellowString("WARNING"), minAWG2Version, routerVersion)
	}
}

// ascParams holds parsed ASC obfuscation parameters from a .conf file.
// All fields are optional — standard WireGuard configs won't have any of these.
type ascParams struct {
	Jc   string
	Jmin string
	Jmax string
	S1   string
	S2   string
	H1   string
	H2   string
	H3   string
	H4   string
	// AWG 2.0
	S3 string
	S4 string
	I1 string
	I2 string
	I3 string
	I4 string
	I5 string
}

// hasAnyASC returns true if at least one AWG 1.0 ASC parameter is present
func (a *ascParams) hasAnyASC() bool {
	return a.Jc != "" || a.Jmin != "" || a.Jmax != "" ||
		a.S1 != "" || a.S2 != "" ||
		a.H1 != "" || a.H2 != "" || a.H3 != "" || a.H4 != ""
}

// hasAWG2 returns true if any AWG 2.0 parameter is present
func (a *ascParams) hasAWG2() bool {
	return a.S3 != "" || a.S4 != "" ||
		a.I1 != "" || a.I2 != "" || a.I3 != "" || a.I4 != "" || a.I5 != ""
}

// peerParams holds parsed peer parameters from a .conf file
type peerParams struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
	PresharedKey        string
}

// parseConfFile reads and parses a WireGuard .conf file, returning ASC and peer parameters.
// ASC parameters are optional (standard WireGuard won't have them).
func parseConfFile(confPath string) (ascParams, peerParams, error) {
	var asc ascParams
	var peer peerParams

	cfg, err := ini.Load(confPath)
	if err != nil {
		return asc, peer, err
	}

	// Parse [Interface] section — ASC params are optional
	interfaceSection, err := cfg.GetSection("Interface")
	if err != nil {
		return asc, peer, fmt.Errorf("conf file missing [Interface] section: %w", err)
	}

	// AWG 1.0 params (all optional)
	if key, err := interfaceSection.GetKey("Jc"); err == nil {
		asc.Jc = key.String()
	}
	if key, err := interfaceSection.GetKey("Jmin"); err == nil {
		asc.Jmin = key.String()
	}
	if key, err := interfaceSection.GetKey("Jmax"); err == nil {
		asc.Jmax = key.String()
	}
	if key, err := interfaceSection.GetKey("S1"); err == nil {
		asc.S1 = key.String()
	}
	if key, err := interfaceSection.GetKey("S2"); err == nil {
		asc.S2 = key.String()
	}
	if key, err := interfaceSection.GetKey("H1"); err == nil {
		asc.H1 = key.String()
	}
	if key, err := interfaceSection.GetKey("H2"); err == nil {
		asc.H2 = key.String()
	}
	if key, err := interfaceSection.GetKey("H3"); err == nil {
		asc.H3 = key.String()
	}
	if key, err := interfaceSection.GetKey("H4"); err == nil {
		asc.H4 = key.String()
	}

	// AWG 2.0 params (optional, KeeneticOS 5.1+)
	if key, err := interfaceSection.GetKey("S3"); err == nil {
		asc.S3 = key.String()
	}
	if key, err := interfaceSection.GetKey("S4"); err == nil {
		asc.S4 = key.String()
	}
	if key, err := interfaceSection.GetKey("I1"); err == nil {
		asc.I1 = key.String()
	}
	if key, err := interfaceSection.GetKey("I2"); err == nil {
		asc.I2 = key.String()
	}
	if key, err := interfaceSection.GetKey("I3"); err == nil {
		asc.I3 = key.String()
	}
	if key, err := interfaceSection.GetKey("I4"); err == nil {
		asc.I4 = key.String()
	}
	if key, err := interfaceSection.GetKey("I5"); err == nil {
		asc.I5 = key.String()
	}

	// Parse [Peer] section
	peerSection, err := cfg.GetSection("Peer")
	if err != nil {
		return asc, peer, fmt.Errorf("conf file missing [Peer] section: %w", err)
	}

	if key, err := peerSection.GetKey("PublicKey"); err == nil {
		peer.PublicKey = key.String()
	}
	if key, err := peerSection.GetKey("Endpoint"); err == nil {
		peer.Endpoint = key.String()
	}
	if key, err := peerSection.GetKey("PresharedKey"); err == nil {
		peer.PresharedKey = key.String()
	}
	if key, err := peerSection.GetKey("PersistentKeepalive"); err == nil {
		val, parseErr := strconv.Atoi(key.String())
		if parseErr == nil {
			peer.PersistentKeepalive = val
		}
	}
	if key, err := peerSection.GetKey("AllowedIPs"); err == nil {
		raw := key.String()
		for cidr := range strings.SplitSeq(raw, ",") {
			cidr = strings.TrimSpace(cidr)
			if cidr != "" {
				peer.AllowedIPs = append(peer.AllowedIPs, cidr)
			}
		}
	}

	return asc, peer, nil
}

// ascNeedsUpdate compares parsed ASC params with current interface state
func ascNeedsUpdate(parsed ascParams, current gokeenrestapimodels.Asc) bool {
	if parsed.Jc != current.Jc || parsed.Jmin != current.Jmin || parsed.Jmax != current.Jmax {
		return true
	}
	if parsed.S1 != current.S1 || parsed.S2 != current.S2 {
		return true
	}
	if parsed.H1 != current.H1 || parsed.H2 != current.H2 || parsed.H3 != current.H3 || parsed.H4 != current.H4 {
		return true
	}
	// AWG 2.0
	if parsed.S3 != current.S3 || parsed.S4 != current.S4 {
		return true
	}
	if parsed.I1 != current.I1 || parsed.I2 != current.I2 || parsed.I3 != current.I3 || parsed.I4 != current.I4 || parsed.I5 != current.I5 {
		return true
	}
	return false
}

// buildASCCommand constructs the RCI command for ASC parameters.
// Format: interface <id> wireguard asc <jc> <jmin> <jmax> <s1> <s2> <h1> <h2> <h3> <h4> [<s3> <s4> <i1> <i2> <i3> <i4> <i5>]
// AWG 2.0 block is all-or-nothing (7 positional params); missing values default to "0".
func buildASCCommand(interfaceId string, asc ascParams) string {
	cmd := fmt.Sprintf("interface %v wireguard asc %v %v %v %v %v %v %v %v %v",
		interfaceId,
		asc.Jc, asc.Jmin, asc.Jmax,
		asc.S1, asc.S2,
		asc.H1, asc.H2, asc.H3, asc.H4)

	// Append AWG 2.0 params if any are present — fill missing with "0"
	if asc.hasAWG2() {
		cmd += fmt.Sprintf(" %v %v %v %v %v %v %v",
			defaultZero(asc.S3), defaultZero(asc.S4),
			defaultZero(asc.I1), defaultZero(asc.I2), defaultZero(asc.I3),
			defaultZero(asc.I4), defaultZero(asc.I5))
	}

	return cmd
}

// defaultZero returns "0" if s is empty, otherwise returns s
func defaultZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// peerNeedsUpdate compares parsed peer params with current interface peer state
func peerNeedsUpdate(parsed peerParams, currentPeers []gokeenrestapimodels.Peer) bool {
	// Find matching peer by public key
	var currentPeer *gokeenrestapimodels.Peer
	for i := range currentPeers {
		if currentPeers[i].Key == parsed.PublicKey {
			currentPeer = &currentPeers[i]
			break
		}
	}

	// If peer not found in current state, it definitely needs update
	if currentPeer == nil {
		return true
	}

	// Compare endpoint
	if parsed.Endpoint != "" && parsed.Endpoint != currentPeer.Endpoint.Address {
		return true
	}

	// Compare keepalive
	if parsed.PersistentKeepalive != currentPeer.KeepaliveInterval.Interval {
		return true
	}

	// Compare preshared key
	if parsed.PresharedKey != currentPeer.PresharedKey {
		return true
	}

	// Compare allowed IPs
	if allowedIPsChanged(parsed.AllowedIPs, currentPeer.AllowIps) {
		return true
	}

	return false
}

// allowedIPsChanged checks if the allowed IP list from conf differs from current state
func allowedIPsChanged(parsed []string, current []gokeenrestapimodels.AllowIps) bool {
	if len(parsed) != len(current) {
		return true
	}

	// Build a set of current allowed IPs as CIDR strings for comparison
	currentSet := make(map[string]struct{}, len(current))
	for _, allowIP := range current {
		cidr := allowIPToCIDR(allowIP)
		currentSet[cidr] = struct{}{}
	}

	for _, parsedCIDR := range parsed {
		normalized := normalizeCIDR(parsedCIDR)
		if _, exists := currentSet[normalized]; !exists {
			return true
		}
	}

	return false
}

// allowIPToCIDR converts a Keenetic AllowIps (address + mask) to CIDR notation
func allowIPToCIDR(allowIP gokeenrestapimodels.AllowIps) string {
	if allowIP.Mask == "" {
		return allowIP.Address + "/32"
	}
	// Convert dotted mask to prefix length
	mask := net.IPMask(net.ParseIP(allowIP.Mask).To4())
	if mask == nil {
		return allowIP.Address + "/" + allowIP.Mask
	}
	ones, _ := mask.Size()
	return fmt.Sprintf("%s/%d", allowIP.Address, ones)
}

// normalizeCIDR normalizes a CIDR string (e.g., removes host bits)
func normalizeCIDR(cidr string) string {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}
	ones, _ := ipNet.Mask.Size()
	return fmt.Sprintf("%s/%d", ipNet.IP.String(), ones)
}

// buildPeerCommands generates RCI commands to update peer configuration
func buildPeerCommands(interfaceId string, parsed peerParams, currentPeers []gokeenrestapimodels.Peer) []gokeenrestapimodels.ParseRequest {
	var commands []gokeenrestapimodels.ParseRequest

	if parsed.PublicKey == "" {
		return commands
	}

	// Find current peer state for comparison
	var currentPeer *gokeenrestapimodels.Peer
	for i := range currentPeers {
		if currentPeers[i].Key == parsed.PublicKey {
			currentPeer = &currentPeers[i]
			break
		}
	}

	// Update endpoint
	if parsed.Endpoint != "" {
		if currentPeer == nil || parsed.Endpoint != currentPeer.Endpoint.Address {
			commands = append(commands, gokeenrestapimodels.ParseRequest{
				Parse: fmt.Sprintf("interface %v wireguard peer %v endpoint %v",
					interfaceId, parsed.PublicKey, parsed.Endpoint),
			})
		}
	}

	// Update keepalive interval
	if currentPeer == nil || parsed.PersistentKeepalive != currentPeer.KeepaliveInterval.Interval {
		commands = append(commands, gokeenrestapimodels.ParseRequest{
			Parse: fmt.Sprintf("interface %v wireguard peer %v keepalive-interval %v",
				interfaceId, parsed.PublicKey, parsed.PersistentKeepalive),
		})
	}

	// Update preshared key
	if parsed.PresharedKey != "" {
		if currentPeer == nil || parsed.PresharedKey != currentPeer.PresharedKey {
			commands = append(commands, gokeenrestapimodels.ParseRequest{
				Parse: fmt.Sprintf("interface %v wireguard peer %v preshared-key %v",
					interfaceId, parsed.PublicKey, parsed.PresharedKey),
			})
		}
	}

	// Update allowed IPs if changed
	if len(parsed.AllowedIPs) > 0 {
		if currentPeer == nil || allowedIPsChanged(parsed.AllowedIPs, currentPeer.AllowIps) {
			// Remove all existing allow-ips first
			if currentPeer != nil && len(currentPeer.AllowIps) > 0 {
				for _, existingIP := range currentPeer.AllowIps {
					cidr := allowIPToCIDR(existingIP)
					commands = append(commands, gokeenrestapimodels.ParseRequest{
						Parse: fmt.Sprintf("no interface %v wireguard peer %v allow-ips %v",
							interfaceId, parsed.PublicKey, cidr),
					})
				}
			}
			// Add new allow-ips
			for _, cidr := range parsed.AllowedIPs {
				commands = append(commands, gokeenrestapimodels.ParseRequest{
					Parse: fmt.Sprintf("interface %v wireguard peer %v allow-ips %v",
						interfaceId, parsed.PublicKey, cidr),
				})
			}
		}
	}

	return commands
}

// ConfigureOrUpdateInterface updates an existing WireGuard interface with configuration from a .conf file.
// It compares the current interface state with the conf file and applies only the necessary changes.
// Supports:
//   - ASC obfuscation parameters (AWG 1.0: Jc, Jmin, Jmax, S1, S2, H1-H4) — optional
//   - AWG 2.0 parameters (S3, S4, I1-I5) — optional, requires KeeneticOS 5.1+
//   - Peer endpoint, allowed IPs, keepalive interval, preshared key
func (*keeneticAwgconf) ConfigureOrUpdateInterface(confPath, interfaceId string) error {
	commands, err := AwgConf.PlanUpdate(confPath, interfaceId)
	if err != nil {
		return err
	}

	if len(commands) == 0 {
		gokeenlog.InfoSubStepf("Interface %v is already up to date", color.CyanString(interfaceId))
		return nil
	}

	return gokeenspinner.WrapWithSpinner(fmt.Sprintf("Updating %v interface configuration", color.CyanString(interfaceId)), func() error {
		commands = Common.EnsureSaveConfigAtEnd(commands)
		_, err := Common.ExecutePostParse(commands...)
		return err
	})
}

// PlanUpdate compares current interface state with a .conf file and returns
// the list of RCI commands that would be executed to bring the interface in sync.
// Returns empty slice if nothing needs to change.
func (*keeneticAwgconf) PlanUpdate(confPath, interfaceId string) ([]gokeenrestapimodels.ParseRequest, error) {
	if confPath == "" {
		return nil, fmt.Errorf("conf-file flag is required")
	}
	if err := Checks.CheckInterfaceId(interfaceId); err != nil {
		return nil, err
	}
	if err := Checks.CheckInterfaceExists(interfaceId); err != nil {
		return nil, err
	}

	var err error
	confPath, err = filepath.Abs(confPath)
	if err != nil {
		return nil, err
	}

	asc, peer, err := parseConfFile(confPath)
	if err != nil {
		return nil, err
	}

	warnIfAWG2Unsupported(asc)

	interfaceDetails, err := Interface.GetInterfaceViaRciShowScInterfaces(interfaceId)
	if err != nil {
		return nil, err
	}

	var parseSlice []gokeenrestapimodels.ParseRequest

	if asc.hasAnyASC() && ascNeedsUpdate(asc, interfaceDetails.Wireguard.Asc) {
		parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
			Parse: buildASCCommand(interfaceId, asc),
		})
	}

	if peer.PublicKey != "" && peerNeedsUpdate(peer, interfaceDetails.Wireguard.Peer) {
		parseSlice = append(parseSlice, buildPeerCommands(interfaceId, peer, interfaceDetails.Wireguard.Peer)...)
	}

	return parseSlice, nil
}

// AddInterface creates a new WireGuard interface from a .conf file
func (*keeneticAwgconf) AddInterface(confFile string, name string) (gokeenrestapimodels.CreatedInterface, error) {
	b, err := os.ReadFile(confFile)
	if err != nil {
		return gokeenrestapimodels.CreatedInterface{}, err
	}
	if name == "" {
		name = filepath.Base(confFile)
	}

	importData := gokeenrestapimodels.Import{Import: base64.StdEncoding.EncodeToString(b), Name: "", Filename: name}

	var createdInterface gokeenrestapimodels.CreatedInterface
	err = gokeenspinner.WrapWithSpinner("Adding interface from the config file", func() error {
		response, err := Common.ExecutePostSubPath("/rci/interface/wireguard/import", importData)
		if err != nil {
			return err
		}
		err = json.Unmarshal(response, &createdInterface)
		if err != nil {
			if response != nil {
				return fmt.Errorf("parsing json error: %v", string(response))
			}
			return err
		}
		for _, status := range createdInterface.Status {
			if status.Status == StatusError {
				err = multierr.Append(err, fmt.Errorf("%v - %v - %v - %v", status.Status, status.Code, status.Ident, status.Message))
			}
		}
		return err
	})
	return createdInterface, err
}

// DiffUpdate returns a unified diff string between current interface state (rendered as .conf)
// and the new conf file (also normalized). Returns empty string if no differences.
func (*keeneticAwgconf) DiffUpdate(confPath, interfaceId string) (string, error) {
	if confPath == "" {
		return "", fmt.Errorf("conf-file flag is required")
	}
	if err := Checks.CheckInterfaceId(interfaceId); err != nil {
		return "", err
	}
	if err := Checks.CheckInterfaceExists(interfaceId); err != nil {
		return "", err
	}

	confPath, err := filepath.Abs(confPath)
	if err != nil {
		return "", err
	}

	// Parse the new conf file into structured params
	asc, peer, err := parseConfFile(confPath)
	if err != nil {
		return "", err
	}

	warnIfAWG2Unsupported(asc)

	// Also get MTU from the new conf (parseConfFile doesn't extract it)
	newMtu := extractMtuFromConf(confPath)

	// Get current state
	interfaceDetails, err := Interface.GetInterfaceViaRciShowScInterfaces(interfaceId)
	if err != nil {
		return "", err
	}

	// Render both sides in canonical .conf format
	currentConf := renderCanonicalConf(interfaceDetails)
	newConf := renderCanonicalConfFromParams(interfaceDetails, asc, peer, newMtu)

	if currentConf == newConf {
		return "", nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(currentConf),
		B:        difflib.SplitLines(newConf),
		FromFile: interfaceId + " (current)",
		ToFile:   filepath.Base(confPath) + " (new)",
		Context:  3,
	}

	return difflib.GetUnifiedDiffString(diff)
}

// extractMtuFromConf reads MTU from a conf file's [Interface] section
func extractMtuFromConf(confPath string) string {
	cfg, err := ini.Load(confPath)
	if err != nil {
		return ""
	}
	section, err := cfg.GetSection("Interface")
	if err != nil {
		return ""
	}
	if key, err := section.GetKey("MTU"); err == nil {
		return key.String()
	}
	return ""
}

// renderCanonicalConf renders current router state as a canonical .conf string
func renderCanonicalConf(sc gokeenrestapimodels.RciShowScInterface) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("Address = %s\n", addressToCIDR(sc.IP.Address)))
	if sc.IP.Mtu != "" && sc.IP.Mtu != "0" {
		b.WriteString(fmt.Sprintf("MTU = %s\n", sc.IP.Mtu))
	}

	asc := sc.Wireguard.Asc
	writeASCBlock(&b, asc)

	for _, peer := range sc.Wireguard.Peer {
		writePeerBlock(&b, peer)
	}

	return b.String()
}

// renderCanonicalConfFromParams renders the "desired" state using parsed params,
// but preserving address/MTU from current if not changed.
func renderCanonicalConfFromParams(current gokeenrestapimodels.RciShowScInterface, asc ascParams, peer peerParams, mtu string) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	// Address comes from current state (we don't change it via update-awg)
	b.WriteString(fmt.Sprintf("Address = %s\n", addressToCIDR(current.IP.Address)))
	// MTU: use new if specified, otherwise keep current
	effectiveMtu := current.IP.Mtu
	if mtu != "" {
		effectiveMtu = mtu
	}
	if effectiveMtu != "" && effectiveMtu != "0" {
		b.WriteString(fmt.Sprintf("MTU = %s\n", effectiveMtu))
	}

	// ASC: use new params if present, otherwise current
	if asc.hasAnyASC() {
		writeASCParamsBlock(&b, asc)
	} else {
		writeASCBlock(&b, current.Wireguard.Asc)
	}

	// Peer: use new params if present, otherwise current
	if peer.PublicKey != "" {
		writePeerParamsBlock(&b, peer)
	} else {
		for _, p := range current.Wireguard.Peer {
			writePeerBlock(&b, p)
		}
	}

	return b.String()
}

// addressToCIDR converts Address model (address + dotted mask) to CIDR notation
func addressToCIDR(addr gokeenrestapimodels.Address) string {
	if addr.Mask == "" {
		return addr.Address
	}
	mask := net.IPMask(net.ParseIP(addr.Mask).To4())
	if mask == nil {
		return addr.Address
	}
	ones, _ := mask.Size()
	return fmt.Sprintf("%s/%d", addr.Address, ones)
}

// writeASCBlock writes ASC parameters from current state in canonical order
func writeASCBlock(b *strings.Builder, asc gokeenrestapimodels.Asc) {
	writeNonZero(b, "Jc", asc.Jc)
	writeNonZero(b, "Jmin", asc.Jmin)
	writeNonZero(b, "Jmax", asc.Jmax)
	writeNonZero(b, "S1", asc.S1)
	writeNonZero(b, "S2", asc.S2)
	writeNonZero(b, "H1", asc.H1)
	writeNonZero(b, "H2", asc.H2)
	writeNonZero(b, "H3", asc.H3)
	writeNonZero(b, "H4", asc.H4)
	writeNonZero(b, "S3", asc.S3)
	writeNonZero(b, "S4", asc.S4)
	writeNonZero(b, "I1", asc.I1)
	writeNonZero(b, "I2", asc.I2)
	writeNonZero(b, "I3", asc.I3)
	writeNonZero(b, "I4", asc.I4)
	writeNonZero(b, "I5", asc.I5)
}

// writeASCParamsBlock writes ASC parameters from parsed conf params in canonical order
func writeASCParamsBlock(b *strings.Builder, asc ascParams) {
	writeNonZero(b, "Jc", asc.Jc)
	writeNonZero(b, "Jmin", asc.Jmin)
	writeNonZero(b, "Jmax", asc.Jmax)
	writeNonZero(b, "S1", asc.S1)
	writeNonZero(b, "S2", asc.S2)
	writeNonZero(b, "H1", asc.H1)
	writeNonZero(b, "H2", asc.H2)
	writeNonZero(b, "H3", asc.H3)
	writeNonZero(b, "H4", asc.H4)
	writeNonZero(b, "S3", asc.S3)
	writeNonZero(b, "S4", asc.S4)
	writeNonZero(b, "I1", asc.I1)
	writeNonZero(b, "I2", asc.I2)
	writeNonZero(b, "I3", asc.I3)
	writeNonZero(b, "I4", asc.I4)
	writeNonZero(b, "I5", asc.I5)
}

// writePeerBlock writes a peer section from current state model
func writePeerBlock(b *strings.Builder, peer gokeenrestapimodels.Peer) {
	b.WriteString("\n[Peer]\n")
	if peer.Key != "" {
		b.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.Key))
	}
	if peer.PresharedKey != "" {
		b.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
	}
	if peer.Endpoint.Address != "" {
		b.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint.Address))
	}
	if len(peer.AllowIps) > 0 {
		cidrs := make([]string, 0, len(peer.AllowIps))
		for _, allowIP := range peer.AllowIps {
			cidrs = append(cidrs, allowIPToCIDR(allowIP))
		}
		b.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(cidrs, ", ")))
	}
	if peer.KeepaliveInterval.Interval > 0 {
		b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.KeepaliveInterval.Interval))
	}
}

// writePeerParamsBlock writes a peer section from parsed conf params
func writePeerParamsBlock(b *strings.Builder, peer peerParams) {
	b.WriteString("\n[Peer]\n")
	b.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
	if peer.PresharedKey != "" {
		b.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
	}
	if peer.Endpoint != "" {
		b.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
	}
	if len(peer.AllowedIPs) > 0 {
		b.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(peer.AllowedIPs, ", ")))
	}
	if peer.PersistentKeepalive > 0 {
		b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
	}
}

// writeNonZero writes a key=value line if value is non-empty and not "0"
func writeNonZero(b *strings.Builder, name, value string) {
	if value != "" && value != "0" {
		b.WriteString(fmt.Sprintf("%s = %s\n", name, value))
	}
}
