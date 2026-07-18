package gokeenrestapi

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

func (m *MockRouter) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	interfaceFilter := r.URL.Query().Get("interface")

	routes := make([]gokeenrestapimodels.RciIpRoute, 0, len(m.routes))
	for _, route := range m.routes {
		if interfaceFilter != "" && route.Interface != interfaceFilter {
			continue
		}
		routes = append(routes, gokeenrestapimodels.RciIpRoute{
			Network:   route.Network,
			Host:      route.Host,
			Mask:      route.Mask,
			Interface: route.Interface,
			Auto:      route.Auto,
		})
	}
	m.encodeJSON(w, routes)
}

func (m *MockRouter) handleShowIpRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	routes := make([]gokeenrestapimodels.RciShowIpRoute, 0, len(m.routes))
	for _, route := range m.routes {
		ip := route.Network
		if route.Host != "" {
			ip = route.Host
		}
		cidr := maskToCIDRString(route.Mask)
		routes = append(routes, gokeenrestapimodels.RciShowIpRoute{
			Destination: fmt.Sprintf("%s/%s", ip, cidr),
			Interface:   route.Interface,
		})
	}
	m.encodeJSON(w, routes)
}

// parseAddRoute handles "ip route <network> <mask> <interface> [auto]" commands.
func (m *MockRouter) parseAddRoute(tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 3 {
		return m.errorResponse("Invalid route command: expected 'ip route <network> <mask> <interface> [auto]'")
	}

	network := tokens[0]
	mask := tokens[1]
	interfaceID := tokens[2]
	auto := len(tokens) >= 4 && tokens[3] == "auto"

	m.mu.RLock()
	_, exists := m.interfaces[interfaceID]
	m.mu.RUnlock()

	if !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.routes = append(m.routes, MockRoute{
		Network:   network,
		Host:      network,
		Mask:      mask,
		Interface: interfaceID,
		Auto:      auto,
	})

	return m.successResponse(fmt.Sprintf("Route %s %s added to interface %s", network, mask, interfaceID))
}

// parseDeleteRoute handles "no ip route <network> <mask> <interface>" commands.
func (m *MockRouter) parseDeleteRoute(tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 3 {
		return m.errorResponse("Invalid route deletion command: expected 'no ip route <network> <mask> <interface>'")
	}

	network := tokens[0]
	mask := tokens[1]
	interfaceID := tokens[2]

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	newRoutes := make([]MockRoute, 0, len(m.routes))
	for _, route := range m.routes {
		if route.Network == network && route.Mask == mask && route.Interface == interfaceID {
			found = true
			continue
		}
		newRoutes = append(newRoutes, route)
	}
	m.routes = newRoutes

	if found {
		return m.successResponse(fmt.Sprintf("Route %s %s removed from interface %s", network, mask, interfaceID))
	}
	return m.successResponse(fmt.Sprintf("Route %s %s on interface %s (not found, but command accepted)", network, mask, interfaceID))
}

// parseAwgConfig handles "interface <id> wireguard asc ..." commands.
// Supports AWG 1.0 (Jc–H4) and AWG 2.0 (S3, S4, I1–I5).
func (m *MockRouter) parseAwgConfig(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 2 || tokens[0] != "asc" {
		return m.errorResponse("Invalid AWG config command: expected 'interface <id> wireguard asc <values...>'")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("SC interface '%s' does not exist", interfaceID))
	}

	v := tokens[1:]
	set := func(idx int, dst *string) {
		if len(v) > idx {
			*dst = v[idx]
		}
	}

	set(0, &scIface.Wireguard.Asc.Jc)
	set(1, &scIface.Wireguard.Asc.Jmin)
	set(2, &scIface.Wireguard.Asc.Jmax)
	set(3, &scIface.Wireguard.Asc.S1)
	set(4, &scIface.Wireguard.Asc.S2)
	set(5, &scIface.Wireguard.Asc.H1)
	set(6, &scIface.Wireguard.Asc.H2)
	set(7, &scIface.Wireguard.Asc.H3)
	set(8, &scIface.Wireguard.Asc.H4)
	set(9, &scIface.Wireguard.Asc.S3)
	set(10, &scIface.Wireguard.Asc.S4)
	set(11, &scIface.Wireguard.Asc.I1)
	set(12, &scIface.Wireguard.Asc.I2)
	set(13, &scIface.Wireguard.Asc.I3)
	set(14, &scIface.Wireguard.Asc.I4)
	set(15, &scIface.Wireguard.Asc.I5)

	return m.successResponse(fmt.Sprintf("AWG parameters updated for interface %s", interfaceID))
}

// parseWireguardPeer handles "interface <id> wireguard peer <key> <subcommand> <value>" commands.
// Supported subcommands: endpoint, keepalive-interval, preshared-key, allow-ips.
func (m *MockRouter) parseWireguardPeer(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 3 {
		return m.errorResponse("Invalid wireguard peer command: expected '<key> <subcommand> <value>'")
	}

	publicKey := tokens[0]
	subcommand := tokens[1]
	value := tokens[2]

	m.mu.Lock()
	defer m.mu.Unlock()

	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("SC interface '%s' does not exist", interfaceID))
	}

	var peer *MockPeer
	for i := range scIface.Wireguard.Peer {
		if scIface.Wireguard.Peer[i].Key == publicKey {
			peer = &scIface.Wireguard.Peer[i]
			break
		}
	}
	if peer == nil {
		scIface.Wireguard.Peer = append(scIface.Wireguard.Peer, MockPeer{Key: publicKey})
		peer = &scIface.Wireguard.Peer[len(scIface.Wireguard.Peer)-1]
	}

	switch subcommand {
	case "endpoint":
		peer.Endpoint = value
	case "keepalive-interval":
		interval, err := strconv.Atoi(value)
		if err != nil {
			return m.errorResponse(fmt.Sprintf("Invalid keepalive-interval value: %s", value))
		}
		peer.KeepaliveInterval = interval
	case "preshared-key":
		peer.PresharedKey = value
	case "allow-ips":
		peer.AllowedIPs = append(peer.AllowedIPs, parseCIDRToAllowedIP(value))
	default:
		return m.errorResponse(fmt.Sprintf("Unknown wireguard peer subcommand: %s", subcommand))
	}

	return m.successResponse(fmt.Sprintf("Wireguard peer %s %s updated for interface %s", publicKey, subcommand, interfaceID))
}

// parseNoWireguardPeer handles "no interface <id> wireguard peer <key> allow-ips <cidr>" commands.
func (m *MockRouter) parseNoWireguardPeer(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 3 {
		return m.errorResponse("Invalid no wireguard peer command: expected '<key> allow-ips <cidr>'")
	}

	publicKey := tokens[0]
	subcommand := tokens[1]
	value := tokens[2]

	if subcommand != "allow-ips" {
		return m.errorResponse(fmt.Sprintf("Unsupported no wireguard peer subcommand: %s", subcommand))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("SC interface '%s' does not exist", interfaceID))
	}

	for i := range scIface.Wireguard.Peer {
		if scIface.Wireguard.Peer[i].Key == publicKey {
			target := parseCIDRToAllowedIP(value)
			newIPs := make([]MockAllowedIP, 0, len(scIface.Wireguard.Peer[i].AllowedIPs))
			for _, allowIP := range scIface.Wireguard.Peer[i].AllowedIPs {
				if allowIP.Address != target.Address || allowIP.Mask != target.Mask {
					newIPs = append(newIPs, allowIP)
				}
			}
			scIface.Wireguard.Peer[i].AllowedIPs = newIPs
			return m.successResponse(fmt.Sprintf("Allow-ips %s removed from peer %s on interface %s", value, publicKey, interfaceID))
		}
	}

	return m.successResponse(fmt.Sprintf("Peer %s not found on interface %s (command accepted)", publicKey, interfaceID))
}

// parseCIDRToAllowedIP converts a CIDR string to MockAllowedIP (address + dotted mask).
func parseCIDRToAllowedIP(cidr string) MockAllowedIP {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		parts := strings.SplitN(cidr, "/", 2)
		if len(parts) == 2 {
			return MockAllowedIP{Address: parts[0], Mask: parts[1]}
		}
		return MockAllowedIP{Address: cidr, Mask: "255.255.255.255"}
	}
	return MockAllowedIP{Address: ipNet.IP.String(), Mask: net.IP(ipNet.Mask).String()}
}
