package exporter

import (
	"encoding/json"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

// dnsUpstreamNames produces at most one display value per existing counter key.
// It never infers loopback mappings from port ranges or array position.
func dnsUpstreamNames(proxy gokeenrestapimodels.MonitoringDNSProxy) map[string]string {
	encrypted := make(map[string][]string)
	for _, source := range []struct {
		body           json.RawMessage
		key, transport string
	}{
		{proxy.TLS, "server-tls", "DoT"}, {proxy.HTTPS, "server-https", "DoH"},
	} {
		var block map[string][]gokeenrestapimodels.MonitoringDNSServer
		if json.Unmarshal(source.body, &block) != nil {
			continue
		}
		for _, server := range block[source.key] {
			origin, name := server.Address, ""
			if source.transport == "DoT" {
				port := "853"
				if server.Port.Valid {
					if server.Port.Value < 1 || server.Port.Value > 65535 || server.Port.Value != float64(int(server.Port.Value)) {
						continue
					}
					port = strconv.Itoa(int(server.Port.Value))
				}
				endpoint, ok := dnsEndpoint(server.Address, port)
				if !ok {
					continue
				}
				name = "DoT " + endpoint
				if safeDNSName(server.SNI) && server.SNI != server.Address {
					name = "DoT " + server.SNI + " (" + endpoint + ")"
				}
			} else {
				if server.URL != "" {
					origin = server.URL
				}
				u, err := url.Parse(origin)
				if err != nil || u.Scheme != "https" || !safeDNSHost(u.Hostname()) {
					continue
				}
				// A path, userinfo, query or fragment may contain a credential.
				name = "DoH " + u.Host
			}
			if !slices.Contains(encrypted[origin], name) {
				encrypted[origin] = append(encrypted[origin], name)
			}
		}
	}
	var configuration string
	if json.Unmarshal(proxy.Configuration, &configuration) != nil {
		return nil
	}
	type metadata struct{ names, domains []string }
	entries := make(map[string]*metadata)
	for line := range strings.SplitSeq(configuration, "\n") {
		parts := strings.SplitN(line, "#", 2)
		fields := strings.Fields(parts[0])
		if len(fields) != 4 || fields[0] != "dns_server" || fields[1] != "=" {
			continue
		}
		endpoint, ok := dnsEndpoint(fields[2], "53")
		if !ok {
			continue
		}
		m := entries[endpoint]
		if m == nil {
			m = &metadata{}
			entries[endpoint] = m
		}
		name := "DNS " + endpoint
		if len(parts) == 2 {
			candidates := encrypted[strings.TrimSpace(parts[1])]
			if len(candidates) != 1 {
				m.names = append(m.names, "")
				continue
			}
			name = candidates[0]
		}
		if !slices.Contains(m.names, name) {
			m.names = append(m.names, name)
		}
		domain := fields[3]
		if domain == "." {
			domain = "default"
		} else if !safeDNSName(domain) {
			continue
		}
		if !slices.Contains(m.domains, domain) {
			m.domains = append(m.domains, domain)
		}
	}
	names := make(map[string]string, len(entries))
	for endpoint, m := range entries {
		if len(m.names) != 1 || m.names[0] == "" {
			continue
		}
		name := m.names[0]
		slices.Sort(m.domains)
		// Bound display size independently of the number of configured suffixes.
		domains := strings.Join(m.domains, ", ")
		if len(domains) > 256 {
			domains = domains[:256] + "…"
		}
		if domains != "" {
			name += " / " + domains
		}
		names[endpoint] = name
	}
	return names
}

func dnsEndpoint(address, defaultPort string) (string, bool) {
	if ip, err := netip.ParseAddr(address); err == nil {
		return net.JoinHostPort(ip.String(), defaultPort), true
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", false
	}
	ip, err := netip.ParseAddr(host)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 1 || n > 65535 {
		return "", false
	}
	return net.JoinHostPort(ip.String(), strconv.Itoa(n)), true
}

func safeDNSHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	return safeDNSName(host)
}

func safeDNSName(name string) bool {
	if name == "" || len(name) > 253 {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune(".-_*", c)) {
			return false
		}
	}
	return true
}
