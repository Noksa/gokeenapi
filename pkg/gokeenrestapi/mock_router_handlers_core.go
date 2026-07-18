package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

// handleAuth implements the authentication endpoint.
// GET requests return 401 with authentication challenge headers.
// POST requests validate credentials and return 200 on success.
func (m *MockRouter) handleAuth(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	realm := m.authRealm
	challenge := m.authChallenge
	cookie := m.sessionCookie
	m.mu.RUnlock()

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("x-ndm-realm", realm)
		w.Header().Set("x-ndm-challenge", challenge)
		w.Header().Set("set-cookie", cookie+"; Path=/")
		w.WriteHeader(http.StatusUnauthorized)
		return

	case http.MethodPost:
		var authRequest struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&authRequest); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if authRequest.Login == "" || authRequest.Password == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockRouter) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	versionStr := m.version
	m.mu.RUnlock()

	version := gokeenrestapimodels.Version{
		Release:      versionStr,
		Title:        versionStr,
		Arch:         "mips",
		Manufacturer: "Keenetic",
		Vendor:       "Keenetic Ltd.",
		Series:       "KN",
		Model:        "KN-1010",
		HwVersion:    "1.0",
		Device:       "Keenetic",
		Description:  "Mock Keenetic Router",
		Ndm: gokeenrestapimodels.Ndm{
			Exact: versionStr,
			Cdate: "2024-01-01",
		},
		Bsp: gokeenrestapimodels.Bsp{
			Exact: versionStr,
			Cdate: "2024-01-01",
		},
		Ndw: gokeenrestapimodels.Ndw{
			Features:   "mock-features",
			Components: "mock-components",
		},
		Ndw4: gokeenrestapimodels.Ndw4{
			Version: "4.0",
		},
	}
	m.encodeJSON(w, version)
}

func (m *MockRouter) handleRunningConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var configLines []string

	if len(m.staticRunningConfig) > 0 {
		configLines = m.staticRunningConfig
	} else {
		configLines = append(configLines, fmt.Sprintf("system mode %s", m.systemMode.Selected))

		for id, iface := range m.interfaces {
			configLines = append(configLines, fmt.Sprintf("interface %s", id))
			if iface.Description != "" {
				configLines = append(configLines, fmt.Sprintf("  description \"%s\"", iface.Description))
			}
			if iface.Address != "" {
				configLines = append(configLines, fmt.Sprintf("  ip address %s", iface.Address))
			}
			if iface.State == StateUp {
				configLines = append(configLines, "  no shutdown")
			} else {
				configLines = append(configLines, "  shutdown")
			}
		}

		for _, route := range m.routes {
			if route.Auto {
				configLines = append(configLines, fmt.Sprintf("ip route %s %s %s auto", route.Network, route.Mask, route.Interface))
			} else {
				configLines = append(configLines, fmt.Sprintf("ip route %s %s %s", route.Network, route.Mask, route.Interface))
			}
		}

		for domain, ip := range m.dnsRecords {
			configLines = append(configLines, fmt.Sprintf("ip host %s %s", domain, ip))
		}

		for _, group := range m.dnsRoutingGroups {
			configLines = append(configLines, fmt.Sprintf("object-group fqdn %s", group.Name))
			for _, domain := range group.Domains {
				configLines = append(configLines, fmt.Sprintf("  include %s", domain))
			}
		}

		for _, route := range m.dnsProxyRoutes {
			configLines = append(configLines, fmt.Sprintf("dns-proxy route object-group %s %s %s", route.GroupName, route.InterfaceID, route.Mode))
		}
	}

	runningConfig := gokeenrestapimodels.RunningConfig{
		Message: configLines,
	}
	m.encodeJSON(w, runningConfig)
}

func (m *MockRouter) handleSystemMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	systemMode := gokeenrestapimodels.SystemMode{
		Active:   m.systemMode.Active,
		Selected: m.systemMode.Selected,
	}
	m.encodeJSON(w, systemMode)
}

func (m *MockRouter) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	override := m.rciBodyOverride
	m.mu.RUnlock()
	if override != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(override)
		return
	}

	var requests []gokeenrestapimodels.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	responses := make([]gokeenrestapimodels.ParseResponse, len(requests))
	for i, req := range requests {
		responses[i] = m.routeParseCommand(req.Parse)
	}
	m.encodeJSON(w, responses)
}

// routeParseCommand parses a command string and dispatches it to the appropriate handler.
// Returns a ParseResponse with either success or error status.
func (m *MockRouter) routeParseCommand(command string) gokeenrestapimodels.ParseResponse {
	command = strings.TrimSpace(command)

	if command == "" {
		return m.successResponse("Empty command accepted")
	}

	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return m.errorResponse("Empty command")
	}

	switch {
	case tokens[0] == "system" && len(tokens) >= 2:
		if tokens[1] == "configuration" && len(tokens) >= 3 && tokens[2] == "save" {
			return m.successResponse("Configuration saved")
		}
		return m.errorResponse(fmt.Sprintf("Unknown system command: %s", strings.Join(tokens[1:], " ")))

	case tokens[0] == "interface" && len(tokens) >= 3:
		return m.dispatchInterfaceCommand(tokens)

	case tokens[0] == "ip" && len(tokens) >= 2:
		return m.dispatchIPCommand(tokens)

	case tokens[0] == "object-group" && len(tokens) >= 3:
		return m.dispatchObjectGroupCommand(tokens)

	case tokens[0] == "dns-proxy" && len(tokens) >= 5:
		return m.dispatchDnsProxyCommand(tokens)

	case tokens[0] == "no" && len(tokens) >= 2:
		return m.dispatchNoCommand(tokens)

	default:
		return m.errorResponse(fmt.Sprintf("Unknown command: %s", tokens[0]))
	}
}

func (m *MockRouter) dispatchInterfaceCommand(tokens []string) gokeenrestapimodels.ParseResponse {
	interfaceID := tokens[1]

	if tokens[2] == "up" || tokens[2] == "down" {
		return m.parseInterfaceState(interfaceID, tokens[2])
	}

	if tokens[2] == "wireguard" && len(tokens) >= 4 {
		if tokens[3] == "asc" {
			return m.parseAwgConfig(interfaceID, tokens[3:])
		}
		if tokens[3] == "peer" && len(tokens) >= 5 {
			return m.parseWireguardPeer(interfaceID, tokens[4:])
		}
		return m.errorResponse(fmt.Sprintf("Unknown wireguard subcommand: %s", tokens[3]))
	}

	if tokens[2] == "create" {
		return m.parseCreateInterface(interfaceID, tokens[3:])
	}

	if tokens[2] == "ip" && len(tokens) >= 5 && tokens[3] == "global" && tokens[4] == "auto" {
		return m.successResponse(fmt.Sprintf("IP global auto enabled for interface %s", interfaceID))
	}

	if tokens[2] == "no" && len(tokens) >= 5 && tokens[3] == "ip" && tokens[4] == "global" {
		return m.successResponse(fmt.Sprintf("IP global disabled for interface %s", interfaceID))
	}

	return m.errorResponse(fmt.Sprintf("Unknown interface command: %s", strings.Join(tokens[2:], " ")))
}

func (m *MockRouter) dispatchIPCommand(tokens []string) gokeenrestapimodels.ParseResponse {
	switch tokens[1] {
	case "route":
		return m.parseAddRoute(tokens[2:])
	case "host":
		return m.parseAddDnsRecord(tokens[2:])
	default:
		return m.errorResponse(fmt.Sprintf("Unknown ip subcommand: %s", tokens[1]))
	}
}

func (m *MockRouter) dispatchObjectGroupCommand(tokens []string) gokeenrestapimodels.ParseResponse {
	if tokens[1] != "fqdn" {
		return m.errorResponse(fmt.Sprintf("Unknown object-group type: %s", tokens[1]))
	}
	groupName := tokens[2]
	if len(tokens) >= 5 && tokens[3] == "include" {
		return m.parseAddDomainToGroup(groupName, tokens[4])
	}
	return m.parseCreateObjectGroup(groupName)
}

func (m *MockRouter) dispatchDnsProxyCommand(tokens []string) gokeenrestapimodels.ParseResponse {
	if tokens[1] == "route" && tokens[2] == "object-group" {
		groupName := tokens[3]
		interfaceID := tokens[4]
		mode := "auto"
		if len(tokens) >= 6 {
			mode = tokens[5]
		}
		return m.parseCreateDnsProxyRoute(groupName, interfaceID, mode)
	}
	return m.errorResponse("Invalid dns-proxy command: expected 'dns-proxy route object-group <group-name> <interface-id> auto'")
}

func (m *MockRouter) dispatchNoCommand(tokens []string) gokeenrestapimodels.ParseResponse {
	switch tokens[1] {
	case "interface":
		if len(tokens) >= 7 && tokens[3] == "wireguard" && tokens[4] == "peer" {
			return m.parseNoWireguardPeer(tokens[2], tokens[5:])
		}
		return m.errorResponse("Invalid no interface command")

	case "object-group":
		if len(tokens) >= 4 && tokens[2] == "fqdn" {
			groupName := tokens[3]
			if len(tokens) >= 6 && tokens[4] == "include" {
				return m.parseRemoveDomainFromGroup(groupName, tokens[5])
			}
			return m.parseDeleteObjectGroup(groupName)
		}
		return m.errorResponse("Invalid object-group deletion: expected 'no object-group fqdn <group-name> [include <domain>]'")

	case "dns-proxy":
		if len(tokens) >= 6 && tokens[2] == "route" && tokens[3] == "object-group" {
			return m.parseDeleteDnsProxyRoute(tokens[4], tokens[5])
		}
		return m.errorResponse("Invalid dns-proxy deletion: expected 'no dns-proxy route object-group <group-name> <interface-id>'")

	case "ip":
		if len(tokens) >= 3 {
			switch tokens[2] {
			case "route":
				return m.parseDeleteRoute(tokens[3:])
			case "host":
				return m.parseDeleteDnsRecord(tokens[3:])
			default:
				return m.errorResponse(fmt.Sprintf("Unknown no ip subcommand: %s", tokens[2]))
			}
		}
		return m.errorResponse("Incomplete no ip command")

	case "known":
		if len(tokens) >= 3 && tokens[2] == "host" {
			return m.parseDeleteKnownHost(tokens[3:])
		}
		return m.errorResponse("Invalid known command: expected 'no known host <mac>'")

	default:
		return m.errorResponse(fmt.Sprintf("Unknown no subcommand: %s", tokens[1]))
	}
}

// errorResponse creates a ParseResponse with error status.
func (m *MockRouter) errorResponse(message string) gokeenrestapimodels.ParseResponse {
	return gokeenrestapimodels.ParseResponse{
		Parse: gokeenrestapimodels.Parse{
			Status: []gokeenrestapimodels.Status{
				{
					Status:  StatusError,
					Code:    "1",
					Message: message,
				},
			},
		},
	}
}

// successResponse creates a ParseResponse with success status.
func (m *MockRouter) successResponse(message string) gokeenrestapimodels.ParseResponse {
	return gokeenrestapimodels.ParseResponse{
		Parse: gokeenrestapimodels.Parse{
			Status: []gokeenrestapimodels.Status{
				{
					Status:  StatusOK,
					Code:    "0",
					Message: message,
				},
			},
		},
	}
}
