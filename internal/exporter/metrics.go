// Package exporter implements cached, read-only Prometheus monitoring for Keenetic routers.
package exporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/multierr"
)

type sample struct {
	name   string
	help   string
	value  float64
	kind   prometheus.ValueType
	labels map[string]string
}

func gauge(name, help string, value float64, labels map[string]string) sample {
	return sample{name: name, help: help, value: value, kind: prometheus.GaugeValue, labels: labels}
}
func counter(name, help string, value float64, labels map[string]string) sample {
	s := gauge(name, help, value, labels)
	s.kind = prometheus.CounterValue
	return s
}

func parseSamples(collector string, bodies []json.RawMessage, interfaces []string) ([]sample, error) {
	if len(bodies) == 0 {
		return nil, errors.New("missing collector response")
	}
	for _, body := range bodies {
		if !json.Valid(body) || string(body) == "null" {
			return nil, errors.New("invalid collector response")
		}
	}
	var samples []sample
	var err error
	switch collector {
	case "system":
		samples, err = parseSystem(bodies[0])
	case "dns":
		samples, err = parseDNS(bodies[0])
	case "hosts":
		samples, err = parseHosts(bodies[0])
	case "info":
		var v gokeenrestapimodels.Version
		if json.Unmarshal(bodies[0], &v) != nil || v.Model == "" || v.Title == "" {
			return nil, errors.New("missing router identity")
		}
		samples = []sample{gauge("keenetic_router_info", "Router model and firmware identity.", 1, map[string]string{"model": v.Model, "version": v.Title})}
	case "interfaces":
		samples, err = parseInterfaces(bodies, interfaces)
	case "static_routes":
		samples, err = parseStaticRoutes(bodies[0])
	case "dns_groups":
		samples, err = parseDNSGroups(bodies[0])
	case "dns_routes":
		samples, err = parseDNSRoutes(bodies[0])
	case "routing":
		if len(bodies) != 3 {
			return nil, errors.New("missing routing responses")
		}
		for i, name := range []string{"static_routes", "dns_groups", "dns_routes"} {
			parsed, e := parseSamples(name, bodies[i:i+1], nil)
			samples = append(samples, parsed...)
			err = multierr.Append(err, e)
		}
	default:
		return nil, errors.New("unknown collector")
	}
	return samples, err
}

func parseSystem(body json.RawMessage) ([]sample, error) {
	var system gokeenrestapimodels.MonitoringSystem
	if err := json.Unmarshal(body, &system); err != nil {
		return nil, errors.New("invalid system statistics")
	}
	var samples []sample
	for _, item := range []struct {
		name, help string
		number     gokeenrestapimodels.MonitoringNumber
		scale      float64
	}{
		{"keenetic_cpu_load_ratio", "Current CPU utilization as a ratio.", system.CPULoad, 0.01},
		{"keenetic_uptime_seconds", "Router uptime in seconds.", system.Uptime, 1},
		{"keenetic_memory_total_bytes", "Total router memory in bytes.", system.MemoryTotal, 1024},
		{"keenetic_memory_free_bytes", "Free router memory in bytes.", system.MemoryFree, 1024},
		{"keenetic_memory_cache_bytes", "Cached router memory in bytes.", system.MemoryCache, 1024},
		{"keenetic_memory_buffers_bytes", "Router buffer memory in bytes.", system.MemoryBuffers, 1024},
		{"keenetic_connections_capacity", "Connection tracking capacity.", system.ConnectionsTotal, 1},
		{"keenetic_connections_free", "Free connection tracking entries.", system.ConnectionsFree, 1},
	} {
		if item.number.Valid {
			samples = append(samples, gauge(item.name, item.help, item.number.Value*item.scale, nil))
		}
	}
	if len(samples) == 0 {
		return nil, errors.New("no supported system statistics")
	}
	if system.MemoryTotal.Valid && system.MemoryFree.Valid && system.MemoryCache.Valid && system.MemoryBuffers.Valid {
		used := system.MemoryTotal.Value - system.MemoryFree.Value - system.MemoryCache.Value - system.MemoryBuffers.Value
		if used >= 0 {
			samples = append(samples, gauge("keenetic_memory_used_bytes", "Used memory excluding cache and buffers, matching Keenetic system memory usage.", used*1024, nil))
		}
	}
	return samples, nil
}

func parseHosts(body json.RawMessage) ([]sample, error) {
	var hosts gokeenrestapimodels.MonitoringHosts
	if json.Unmarshal(body, &hosts) != nil || hosts.Hosts == nil {
		return nil, errors.New("missing host list")
	}
	samples := []sample{gauge("keenetic_hosts", "Known devices, including inactive devices.", float64(len(hosts.Hosts)), nil)}
	active, registered := 0, 0
	activityKnown, registrationKnown := true, true
	for _, host := range hosts.Hosts {
		if host.Active != nil {
			if *host.Active {
				active++
			}
		} else if host.Link != "" {
			if host.Link == "up" {
				active++
			}
		} else {
			activityKnown = false
		}
		if host.Registered != nil {
			if *host.Registered {
				registered++
			}
		} else {
			registrationKnown = false
		}
	}
	if activityKnown {
		samples = append(samples, gauge("keenetic_hosts_active", "Currently active known devices.", float64(active), nil))
	}
	if registrationKnown {
		samples = append(samples, gauge("keenetic_hosts_registered", "Registered known devices.", float64(registered), nil))
	}
	return samples, nil
}

func parseStaticRoutes(body json.RawMessage) ([]sample, error) {
	var routes []gokeenrestapimodels.RciIpRoute
	if json.Unmarshal(body, &routes) != nil || routes == nil {
		return nil, errors.New("invalid configured static routes")
	}
	counts := make(map[string]int)
	for _, r := range routes {
		if r.Host == "" && r.Network == "" {
			return nil, errors.New("missing static route destination")
		}
		counts[r.Interface]++
	}
	samples := []sample{gauge("keenetic_static_routes_count", "Total configured static routes.", float64(len(routes)), nil)}
	for id, n := range counts {
		samples = append(samples, gauge("keenetic_static_routes", "Configured static routes per outgoing interface.", float64(n), map[string]string{"interface": id}))
	}
	return samples, nil
}

func parseDNSGroups(body json.RawMessage) ([]sample, error) {
	var groups gokeenrestapimodels.ObjectGroupFqdnResponse
	if json.Unmarshal(body, &groups) != nil || groups == nil {
		return nil, errors.New("invalid DNS object groups")
	}
	samples := []sample{gauge("keenetic_dns_groups", "Configured FQDN object groups.", float64(len(groups)), nil)}
	for name, group := range groups {
		counts := map[string]int{"domain": 0, "ip": 0}
		for _, entry := range group.Include {
			if entry.Address == "" {
				return nil, errors.New("missing DNS group entry")
			}
			kind := "domain"
			if _, err := netip.ParseAddr(entry.Address); err == nil {
				kind = "ip"
			} else if _, err := netip.ParsePrefix(entry.Address); err == nil {
				kind = "ip"
			}
			counts[kind]++
		}
		for kind, n := range counts {
			samples = append(samples, gauge("keenetic_dns_group_entries", "Configured DNS group entries; IP literals are counted separately from domains.", float64(n), map[string]string{"group": name, "entry_type": kind}))
		}
	}
	return samples, nil
}

func parseDNSRoutes(body json.RawMessage) ([]sample, error) {
	var routes gokeenrestapimodels.DnsProxyRouteResponse
	if json.Unmarshal(body, &routes) != nil || routes == nil {
		return nil, errors.New("invalid DNS proxy route list")
	}
	samples := []sample{gauge("keenetic_dns_routes_count", "Total configured system DNS-routing rules.", float64(len(routes)), nil)}
	counts := make(map[[2]string]int)
	for _, r := range routes {
		if r.Group == "" || r.Interface == "" {
			return nil, errors.New("missing DNS route group or interface")
		}
		counts[[2]string{r.Group, r.Interface}]++
	}
	for key, n := range counts {
		samples = append(samples, gauge("keenetic_dns_routes", "Configured system DNS-routing rules per group and interface.", float64(n), map[string]string{"group": key[0], "interface": key[1]}))
	}
	return samples, nil
}

func selectedInterfaces(body json.RawMessage, filter []string) ([]string, error) {
	interfaces, err := monitoringInterfaces(body)
	if err != nil {
		return nil, err
	}
	var selected []string
	for id, iface := range interfaces {
		if iface.Id == "" {
			continue
		}
		if (len(filter) > 0 && slices.Contains(filter, id)) || (len(filter) == 0 && (iface.Address != "" || iface.DefaultGw || strings.EqualFold(iface.Type, "Wireguard"))) {
			selected = append(selected, id)
		}
	}
	slices.Sort(selected)
	if len(filter) > 0 {
		for _, id := range filter {
			if !slices.Contains(selected, id) {
				return nil, fmt.Errorf("configured interface %q was not returned by router", id)
			}
		}
	}
	return selected, nil
}

func parseInterfaces(bodies []json.RawMessage, ids []string) ([]sample, error) {
	interfaces, err := monitoringInterfaces(bodies[0])
	if err != nil {
		return nil, err
	}
	if len(bodies) != len(ids)+1 {
		return nil, errors.New("missing interface counter responses")
	}
	var samples []sample
	for i, id := range ids {
		iface, ok := interfaces[id]
		if !ok {
			return nil, errors.New("missing selected interface")
		}
		labels := map[string]string{"interface": id}
		if iface.State != "" {
			up := 0.0
			if iface.State == "up" {
				up = 1
			}
			samples = append(samples, gauge("keenetic_interface_up", "Administrative interface state.", up, labels))
		}
		if iface.Connected != "" {
			up := 0.0
			if iface.Connected == "yes" {
				up = 1
			}
			samples = append(samples, gauge("keenetic_interface_connected", "Operational interface connectivity.", up, labels))
		}
		var stat gokeenrestapimodels.MonitoringInterfaceStat
		if json.Unmarshal(bodies[i+1], &stat) != nil {
			return nil, errors.New("invalid interface statistics")
		}
		for _, item := range []struct {
			name, help string
			number     gokeenrestapimodels.MonitoringNumber
		}{
			{"keenetic_interface_receive_bytes_total", "Bytes received on the interface.", stat.RXBytes},
			{"keenetic_interface_transmit_bytes_total", "Bytes transmitted on the interface.", stat.TXBytes},
			{"keenetic_interface_receive_packets_total", "Packets received on the interface.", stat.RXPackets},
			{"keenetic_interface_transmit_packets_total", "Packets transmitted on the interface.", stat.TXPackets},
			{"keenetic_interface_receive_errors_total", "Receive errors on the interface.", stat.RXErrors},
			{"keenetic_interface_transmit_errors_total", "Transmit errors on the interface.", stat.TXErrors},
			{"keenetic_interface_receive_dropped_total", "Receive drops on the interface.", stat.RXDropped},
			{"keenetic_interface_transmit_dropped_total", "Transmit drops on the interface.", stat.TXDropped},
		} {
			if item.number.Valid {
				samples = append(samples, counter(item.name, item.help, item.number.Value, labels))
			}
		}
	}
	return samples, nil
}
