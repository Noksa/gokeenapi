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
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"go.uber.org/multierr"
	"gopkg.in/ini.v1"
)

var (
	// AwgConf provides WireGuard (AWG) configuration management functionality
	AwgConf keeneticAwgconf
)

type keeneticAwgconf struct{}

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
	if confPath == "" {
		return fmt.Errorf("conf-file flag is required")
	}
	err := Checks.CheckInterfaceId(interfaceId)
	if err != nil {
		return err
	}
	err = Checks.CheckInterfaceExists(interfaceId)
	if err != nil {
		return err
	}

	confPath, err = filepath.Abs(confPath)
	if err != nil {
		return err
	}

	// Parse conf file
	asc, peer, err := parseConfFile(confPath)
	if err != nil {
		return err
	}

	// Get current interface state
	interfaceDetails, err := Interface.GetInterfaceViaRciShowScInterfaces(interfaceId)
	if err != nil {
		return err
	}

	var parseSlice []gokeenrestapimodels.ParseRequest

	// Build ASC command if ASC parameters are present and changed
	if asc.hasAnyASC() {
		if ascNeedsUpdate(asc, interfaceDetails.Wireguard.Asc) {
			parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
				Parse: buildASCCommand(interfaceId, asc),
			})
		}
	}

	// Build peer update commands if peer configuration has changed
	if peer.PublicKey != "" && peerNeedsUpdate(peer, interfaceDetails.Wireguard.Peer) {
		peerCmds := buildPeerCommands(interfaceId, peer, interfaceDetails.Wireguard.Peer)
		parseSlice = append(parseSlice, peerCmds...)
	}

	// Nothing to update
	if len(parseSlice) == 0 {
		gokeenlog.InfoSubStepf("Interface %v is already up to date", color.CyanString(interfaceId))
		return nil
	}

	return gokeenspinner.WrapWithSpinner(fmt.Sprintf("Updating %v interface configuration", color.CyanString(interfaceId)), func() error {
		parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)
		_, err := Common.ExecutePostParse(parseSlice...)
		return err
	})
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
