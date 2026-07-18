package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

func (m *MockRouter) handleDnsRecords(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	response := map[string]map[string]string{
		"static": m.dnsRecords,
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleObjectGroupFqdn(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	response := make(gokeenrestapimodels.ObjectGroupFqdnResponse)
	for _, group := range m.dnsRoutingGroups {
		entries := make([]gokeenrestapimodels.ObjectGroupFqdnEntry, 0, len(group.Domains))
		for _, domain := range group.Domains {
			entries = append(entries, gokeenrestapimodels.ObjectGroupFqdnEntry{Address: domain})
		}
		response[group.Name] = gokeenrestapimodels.ObjectGroupFqdn{Include: entries}
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleDnsProxyRoute(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	routes := make(gokeenrestapimodels.DnsProxyRouteResponse, 0, len(m.dnsProxyRoutes))
	for _, route := range m.dnsProxyRoutes {
		routes = append(routes, gokeenrestapimodels.DnsProxyRoute{
			Group:     route.GroupName,
			Interface: route.InterfaceID,
		})
	}
	m.encodeJSON(w, routes)
}

// handleComponentsList handles POST /rci/components/list requests.
func (m *MockRouter) handleComponentsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	components := make(map[string]gokeenrestapimodels.Component, len(m.components))
	maps.Copy(components, m.components)
	m.mu.RUnlock()

	result := gokeenrestapimodels.RciComponentsList{
		Sandbox:   "mock-sandbox",
		Component: components,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// parseAddDnsRecord handles "ip host <domain> <ip>" commands.
func (m *MockRouter) parseAddDnsRecord(tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 2 {
		return m.errorResponse("Invalid DNS command: expected 'ip host <domain> <ip>'")
	}

	domain := tokens[0]
	ip := tokens[1]

	m.mu.Lock()
	defer m.mu.Unlock()

	m.dnsRecords[domain] = ip
	return m.successResponse(fmt.Sprintf("DNS record %s -> %s added", domain, ip))
}

// parseDeleteDnsRecord handles "no ip host <domain>" commands.
func (m *MockRouter) parseDeleteDnsRecord(tokens []string) gokeenrestapimodels.ParseResponse {
	if len(tokens) < 1 {
		return m.errorResponse("Invalid DNS deletion command: expected 'no ip host <domain>'")
	}

	domain := tokens[0]

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.dnsRecords, domain)
	return m.successResponse(fmt.Sprintf("DNS record for domain '%s' removed", domain))
}

// parseCreateObjectGroup handles "object-group fqdn <group-name>" commands.
func (m *MockRouter) parseCreateObjectGroup(groupName string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group command: group name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			return m.successResponse(fmt.Sprintf("Object-group '%s' already exists", groupName))
		}
	}

	m.dnsRoutingGroups = append(m.dnsRoutingGroups, MockDnsRoutingGroup{
		Name:    groupName,
		Domains: []string{},
	})
	return m.successResponse(fmt.Sprintf("Object-group '%s' created", groupName))
}

// parseAddDomainToGroup handles "object-group fqdn <group-name> include <domain>" commands.
func (m *MockRouter) parseAddDomainToGroup(groupName, domain string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group command: group name cannot be empty")
	}
	if domain == "" {
		return m.errorResponse("Invalid object-group command: domain cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.dnsRoutingGroups {
		if m.dnsRoutingGroups[i].Name == groupName {
			m.dnsRoutingGroups[i].Domains = append(m.dnsRoutingGroups[i].Domains, domain)
			return m.successResponse(fmt.Sprintf("Domain '%s' added to object-group '%s'", domain, groupName))
		}
	}
	return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
}

// parseCreateDnsProxyRoute handles "dns-proxy route object-group <group-name> <interface-id> auto" commands.
func (m *MockRouter) parseCreateDnsProxyRoute(groupName, interfaceID, mode string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid dns-proxy route command: group name cannot be empty")
	}
	if interfaceID == "" {
		return m.errorResponse("Invalid dns-proxy route command: interface ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.interfaces[interfaceID]; !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	groupExists := false
	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			groupExists = true
			break
		}
	}
	if !groupExists {
		return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
	}

	for _, route := range m.dnsProxyRoutes {
		if route.GroupName == groupName && route.InterfaceID == interfaceID {
			return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' already exists", groupName, interfaceID))
		}
	}

	m.dnsProxyRoutes = append(m.dnsProxyRoutes, MockDnsProxyRoute{
		GroupName:   groupName,
		InterfaceID: interfaceID,
		Mode:        mode,
	})
	return m.successResponse(fmt.Sprintf("Dns-proxy route created for group '%s' to interface '%s'", groupName, interfaceID))
}

// parseDeleteDnsProxyRoute handles "no dns-proxy route object-group <group-name> <interface-id>" commands.
func (m *MockRouter) parseDeleteDnsProxyRoute(groupName, interfaceID string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid dns-proxy route deletion: group name cannot be empty")
	}
	if interfaceID == "" {
		return m.errorResponse("Invalid dns-proxy route deletion: interface ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	newRoutes := make([]MockDnsProxyRoute, 0, len(m.dnsProxyRoutes))
	for _, route := range m.dnsProxyRoutes {
		if route.GroupName == groupName && route.InterfaceID == interfaceID {
			found = true
			continue
		}
		newRoutes = append(newRoutes, route)
	}
	m.dnsProxyRoutes = newRoutes

	if found {
		return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' removed", groupName, interfaceID))
	}
	return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' (not found, but command accepted)", groupName, interfaceID))
}

// parseRemoveDomainFromGroup handles "no object-group fqdn <group-name> include <domain>" commands.
func (m *MockRouter) parseRemoveDomainFromGroup(groupName, domain string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid domain removal: group name cannot be empty")
	}
	if domain == "" {
		return m.errorResponse("Invalid domain removal: domain cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.dnsRoutingGroups {
		if m.dnsRoutingGroups[i].Name == groupName {
			found := false
			newDomains := make([]string, 0, len(m.dnsRoutingGroups[i].Domains))
			for _, d := range m.dnsRoutingGroups[i].Domains {
				if d == domain {
					found = true
					continue
				}
				newDomains = append(newDomains, d)
			}
			m.dnsRoutingGroups[i].Domains = newDomains

			if found {
				return m.successResponse(fmt.Sprintf("Domain '%s' removed from object-group '%s'", domain, groupName))
			}
			return m.successResponse(fmt.Sprintf("Domain '%s' not found in object-group '%s' (command accepted)", domain, groupName))
		}
	}
	return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
}

// parseDeleteObjectGroup handles "no object-group fqdn <group-name>" commands.
func (m *MockRouter) parseDeleteObjectGroup(groupName string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group deletion: group name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	newGroups := make([]MockDnsRoutingGroup, 0, len(m.dnsRoutingGroups))
	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			found = true
			continue
		}
		newGroups = append(newGroups, group)
	}
	m.dnsRoutingGroups = newGroups

	if found {
		return m.successResponse(fmt.Sprintf("Object-group '%s' removed", groupName))
	}
	return m.successResponse(fmt.Sprintf("Object-group '%s' (not found, but command accepted)", groupName))
}
