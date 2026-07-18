package gokeenrestapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

func (m *MockRouter) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	typeFilter := r.URL.Query().Get("type")

	interfaces := make(map[string]gokeenrestapimodels.RciShowInterface)
	for id, iface := range m.interfaces {
		if typeFilter != "" && iface.Type != typeFilter {
			continue
		}
		interfaces[id] = gokeenrestapimodels.RciShowInterface{
			Id:          iface.ID,
			Type:        iface.Type,
			Description: iface.Description,
			Address:     iface.Address,
			Connected:   iface.Connected,
			Link:        iface.Link,
			State:       iface.State,
			DefaultGw:   iface.DefaultGw,
		}
	}
	m.encodeJSON(w, interfaces)
}

func (m *MockRouter) handleInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaceID := strings.TrimPrefix(r.URL.Path, "/rci/show/interface/")

	m.mu.RLock()
	defer m.mu.RUnlock()

	iface, exists := m.interfaces[interfaceID]
	if !exists {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	response := gokeenrestapimodels.RciShowInterface{
		Id:          iface.ID,
		Type:        iface.Type,
		Description: iface.Description,
		Address:     iface.Address,
		Connected:   iface.Connected,
		Link:        iface.Link,
		State:       iface.State,
		DefaultGw:   iface.DefaultGw,
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleScInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	interfaces := make(map[string]gokeenrestapimodels.RciShowScInterface)
	for id, scIface := range m.scInterfaces {
		interfaces[id] = convertScInterface(scIface)
	}
	m.encodeJSON(w, interfaces)
}

func (m *MockRouter) handleScInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaceID := strings.TrimPrefix(r.URL.Path, "/rci/show/sc/interface/")

	m.mu.RLock()
	defer m.mu.RUnlock()

	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	m.encodeJSON(w, convertScInterface(scIface))
}

func (m *MockRouter) handleHotspot(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hosts := make([]gokeenrestapimodels.Host, 0, len(m.hotspotDevices))
	for _, device := range m.hotspotDevices {
		hosts = append(hosts, gokeenrestapimodels.Host{
			Name:       device.Name,
			Mac:        device.Mac,
			IP:         device.IP,
			Hostname:   device.Hostname,
			Registered: device.Registered,
			Link:       device.Link,
			Via:        device.Via,
		})
	}
	m.encodeJSON(w, gokeenrestapimodels.RciShowIpHotspot{Host: hosts})
}

// convertScInterface converts MockScInterface to the API response type.
// Extracted to avoid duplication between handleScInterfaces and handleScInterface.
func convertScInterface(scIface *MockScInterface) gokeenrestapimodels.RciShowScInterface {
	peers := make([]gokeenrestapimodels.Peer, len(scIface.Wireguard.Peer))
	for i, mockPeer := range scIface.Wireguard.Peer {
		allowIps := make([]gokeenrestapimodels.AllowIps, len(mockPeer.AllowedIPs))
		for j, mockAllowedIP := range mockPeer.AllowedIPs {
			allowIps[j] = gokeenrestapimodels.AllowIps{
				Address: mockAllowedIP.Address,
				Mask:    mockAllowedIP.Mask,
			}
		}
		peers[i] = gokeenrestapimodels.Peer{
			Key:     mockPeer.Key,
			Comment: mockPeer.Comment,
			Endpoint: gokeenrestapimodels.Endpoint{
				Address: mockPeer.Endpoint,
			},
			KeepaliveInterval: gokeenrestapimodels.KeepaliveInterval{
				Interval: mockPeer.KeepaliveInterval,
			},
			PresharedKey: mockPeer.PresharedKey,
			AllowIps:     allowIps,
		}
	}

	return gokeenrestapimodels.RciShowScInterface{
		Description: scIface.Description,
		IP: gokeenrestapimodels.IP{
			Address: gokeenrestapimodels.Address{
				Address: scIface.IP.Address,
			},
		},
		Wireguard: gokeenrestapimodels.Wireguard{
			Asc: gokeenrestapimodels.Asc{
				Jc:   scIface.Wireguard.Asc.Jc,
				Jmin: scIface.Wireguard.Asc.Jmin,
				Jmax: scIface.Wireguard.Asc.Jmax,
				S1:   scIface.Wireguard.Asc.S1,
				S2:   scIface.Wireguard.Asc.S2,
				H1:   scIface.Wireguard.Asc.H1,
				H2:   scIface.Wireguard.Asc.H2,
				H3:   scIface.Wireguard.Asc.H3,
				H4:   scIface.Wireguard.Asc.H4,
				S3:   scIface.Wireguard.Asc.S3,
				S4:   scIface.Wireguard.Asc.S4,
				I1:   scIface.Wireguard.Asc.I1,
				I2:   scIface.Wireguard.Asc.I2,
				I3:   scIface.Wireguard.Asc.I3,
				I4:   scIface.Wireguard.Asc.I4,
				I5:   scIface.Wireguard.Asc.I5,
			},
			Peer: peers,
		},
	}
}

// parseInterfaceState handles "interface <id> up|down" commands.
func (m *MockRouter) parseInterfaceState(interfaceID string, state string) gokeenrestapimodels.ParseResponse {
	if state != "up" && state != "down" {
		return m.errorResponse(fmt.Sprintf("Invalid interface state: %s (expected 'up' or 'down')", state))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	iface, exists := m.interfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	if state == "up" {
		iface.Connected = StateConnected
		iface.Link = StateUp
		iface.State = StateUp
	} else {
		iface.Connected = StateDisconnected
		iface.Link = StateDown
		iface.State = StateDown
	}

	return m.successResponse(fmt.Sprintf("Interface %s set to %s", interfaceID, state))
}

// parseCreateInterface handles "interface <id> create type <type> ..." commands.
func (m *MockRouter) parseCreateInterface(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 2 || tokens[0] != "type" {
		return m.errorResponse("Invalid interface creation command: expected 'interface <id> create type <type> ...'")
	}

	interfaceType := tokens[1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.interfaces[interfaceID]; exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' already exists", interfaceID))
	}

	description := ""
	address := ""
	for i := 2; i < len(tokens)-1; i += 2 {
		switch strings.ToLower(tokens[i]) {
		case "description":
			description = tokens[i+1]
		case "address":
			address = tokens[i+1]
		}
	}

	m.interfaces[interfaceID] = &MockInterface{
		ID:          interfaceID,
		Type:        interfaceType,
		Description: description,
		Address:     address,
		Connected:   StateDisconnected,
		Link:        StateDown,
		State:       StateDown,
		DefaultGw:   false,
	}

	if interfaceType == InterfaceTypeWireguard {
		m.scInterfaces[interfaceID] = &MockScInterface{
			Description: description,
			IP:          MockIP{Address: address},
			Wireguard: MockWireguard{
				Asc:  MockAsc{Jc: "0", Jmin: "0", Jmax: "0", S1: "0", S2: "0", H1: "0", H2: "0", H3: "0", H4: "0"},
				Peer: []MockPeer{},
			},
		}
	}

	return m.successResponse(fmt.Sprintf("Interface %s created with type %s", interfaceID, interfaceType))
}

// parseDeleteKnownHost handles "no known host <mac>" commands.
func (m *MockRouter) parseDeleteKnownHost(tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 1 {
		return m.errorResponse("Invalid known host deletion command: expected 'no known host \"<mac>\"'")
	}

	mac := strings.Trim(tokens[0], "\"")

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	newHosts := make([]MockHost, 0, len(m.hotspotDevices))
	for _, host := range m.hotspotDevices {
		if strings.EqualFold(host.Mac, mac) {
			found = true
			continue
		}
		newHosts = append(newHosts, host)
	}
	m.hotspotDevices = newHosts

	if found {
		return m.successResponse(fmt.Sprintf("Known host with MAC '%s' removed", mac))
	}
	return m.successResponse(fmt.Sprintf("Known host with MAC '%s' (not found, but command accepted)", mac))
}
