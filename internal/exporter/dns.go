package exporter

import (
	"encoding/json"
	"errors"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

var dnsRequests = regexp.MustCompile(`(?m)^\s*Total incoming requests:\s*(\d+)\s*$`)
var dnsSent = regexp.MustCompile(`(?m)^\s*Proxy requests sent:\s*(\d+)\s*$`)
var dnsHits = regexp.MustCompile(`(?m)^\s*Cache hits ratio:\s*[\d.]+\s*\((\d+)\)\s*$`)

func parseDNS(body json.RawMessage) ([]sample, error) {
	var proxies gokeenrestapimodels.MonitoringDNS
	if json.Unmarshal(body, &proxies) != nil || len(proxies.Proxies) == 0 {
		return nil, errors.New("missing DNS proxy statistics")
	}
	var samples []sample
	seen := make(map[string]bool)
	for _, proxy := range proxies.Proxies {
		if proxy.Name == "" || seen[proxy.Name] {
			return nil, errors.New("missing or duplicate DNS proxy name")
		}
		seen[proxy.Name] = true
		before := len(samples)
		labels := map[string]string{"proxy": proxy.Name}
		for _, field := range []struct {
			pattern    *regexp.Regexp
			name, help string
		}{
			{dnsRequests, "keenetic_dns_requests_total", "DNS requests received by this proxy."},
			{dnsSent, "keenetic_dns_forwarded_requests_total", "DNS requests sent to upstream resolvers."},
			{dnsHits, "keenetic_dns_cache_hits_total", "DNS requests answered from the proxy cache."},
		} {
			match := field.pattern.FindStringSubmatch(proxy.Statistics)
			if len(match) > 1 {
				value, err := strconv.ParseFloat(match[1], 64)
				if err != nil {
					return nil, errors.New("invalid DNS counter")
				}
				samples = append(samples, counter(field.name, field.help, value, labels))
			}
		}
		if len(samples) == before {
			return nil, errors.New("unrecognized DNS proxy statistics")
		}
		upstreams := make(map[[2]string]bool)
		for line := range strings.SplitSeq(proxy.Statistics, "\n") {
			fields := strings.Fields(line)
			if len(fields) != 8 {
				continue
			}
			if _, err := netip.ParseAddr(fields[0]); err != nil {
				continue
			}
			port, err := strconv.Atoi(fields[1])
			if err != nil || port < 1 || port > 65535 {
				continue
			}
			key := [2]string{fields[0], fields[1]}
			if upstreams[key] {
				return nil, errors.New("duplicate DNS upstream row")
			}
			upstreams[key] = true
			labels := map[string]string{"proxy": proxy.Name, "server": fields[0], "port": fields[1]}
			for i, field := range []struct{ name, help string }{
				{"keenetic_dns_upstream_requests_total", "DNS requests sent to an upstream resolver."},
				{"keenetic_dns_upstream_responses_total", "Answers received from an upstream resolver."},
				{"keenetic_dns_upstream_nxdomain_total", "NX answers received from an upstream resolver."},
			} {
				value, err := strconv.ParseUint(fields[i+2], 10, 64)
				if err != nil {
					return nil, errors.New("invalid DNS upstream counter")
				}
				samples = append(samples, counter(field.name, field.help, float64(value), labels))
			}
			if duration, err := time.ParseDuration(fields[6]); err == nil && duration >= 0 {
				samples = append(samples, gauge("keenetic_dns_upstream_response_duration_seconds", "Router-reported average upstream response time, not a histogram.", duration.Seconds(), labels))
			}
		}
	}
	return samples, nil
}
