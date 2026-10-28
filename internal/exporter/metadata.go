package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

const policyRefresh = 5 * time.Minute
const policyMaxAge = 2 * policyRefresh

func displayName(id string, candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && candidate != id {
			return candidate + " (" + id + ")"
		}
	}
	return id
}

func monitoringInterfaces(body json.RawMessage) (map[string]gokeenrestapimodels.MonitoringInterface, error) {
	var raw map[string]gokeenrestapimodels.MonitoringInterface
	if json.Unmarshal(body, &raw) != nil || raw == nil {
		return nil, errors.New("invalid interface list")
	}
	interfaces := make(map[string]gokeenrestapimodels.MonitoringInterface, len(raw))
	for _, iface := range raw {
		if iface.Id == "" {
			continue
		}
		if _, duplicate := interfaces[iface.Id]; duplicate {
			return nil, errors.New("duplicate interface ID")
		}
		interfaces[iface.Id] = iface
	}
	return interfaces, nil
}

func (t *target) rememberInterfaces(body json.RawMessage) ([]sample, error) {
	interfaces, err := monitoringInterfaces(body)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(interfaces))
	var speeds []sample
	for id, iface := range interfaces {
		var name string
		_ = json.Unmarshal(iface.InterfaceName, &name) // Unsupported optional metadata falls back to ID.
		names[id] = displayName(id, iface.Description, name)
		if len(t.interfaces) > 0 && !slices.Contains(t.interfaces, id) {
			continue
		}
		var speed gokeenrestapimodels.MonitoringNumber
		if iface.Link == "up" && json.Unmarshal(iface.Speed, &speed) == nil && speed.Valid && speed.Value > 0 {
			speeds = append(speeds, gauge("keenetic_interface_speed_bits_per_second", "Negotiated physical link speed in bits per second, when reported by the router.", speed.Value*1e6, map[string]string{"interface": id}))
		}
	}
	t.mu.Lock()
	t.interfaceNames, t.interfaceNamesAt = names, time.Now()
	t.mu.Unlock()
	return speeds, nil
}

// Policy metadata is optional: failures do not invalidate DNS statistics. Reads
// share the router's serialized poll and timeout, with bounded retry and expiry.
func (t *target) rememberDNSPolicies(body json.RawMessage) {
	var dns gokeenrestapimodels.MonitoringDNS
	if json.Unmarshal(body, &dns) != nil {
		return
	}
	t.mu.Lock()
	t.namedPolicies = slices.ContainsFunc(dns.Proxies, func(p gokeenrestapimodels.MonitoringDNSProxy) bool { return p.Name != "" && p.Name != "System" })
	t.mu.Unlock()
}

func (e *Exporter) refreshPolicies(ctx context.Context, t *target, now time.Time) {
	t.mu.RLock()
	due := t.namedPolicies && !now.Before(t.policyNext)
	t.mu.RUnlock()
	if !due {
		return
	}
	result := t.session.Read(ctx, []gokeenrestapi.MonitoringQuery{{Path: "/rci/ip/policy"}})[0]
	var policies map[string]gokeenrestapimodels.MonitoringPolicy
	err := result.Err
	if err == nil && (json.Unmarshal(result.Data, &policies) != nil || policies == nil) {
		err = errors.New("invalid policy metadata")
	}
	scheduleBase := now
	if finished := time.Now(); finished.After(scheduleBase) {
		scheduleBase = finished
	}
	if err != nil {
		t.mu.Lock()
		firstFailure := t.policyFailures == 0
		t.policyFailures++
		t.policyNext = scheduleBase.Add(retryInterval(policyRefresh, t.policyFailures, e.cfg.MaxBackoff))
		t.mu.Unlock()
		if firstFailure {
			gokeenlog.InfoSubStepf("Router %q: policy names unavailable; using IDs", t.name)
		}
		return
	}
	names := make(map[string]string, len(policies))
	for id, policy := range policies {
		names[id] = displayName(id, policy.Description)
	}
	t.mu.Lock()
	t.policyNames, t.policyNamesAt = names, time.Now()
	t.policyFailures = 0
	t.policyNext = scheduleBase.Add(policyRefresh)
	t.mu.Unlock()
}

// Called under the target read lock; cached samples and label maps stay immutable.
func (t *target) namedSample(s sample, now time.Time) sample {
	id, hasInterface := s.labels["interface"]
	_, hasProxy := s.labels["proxy"]
	if !hasInterface && !hasProxy {
		return s
	}
	s.labels = maps.Clone(s.labels)
	if hasInterface {
		name := id
		if id == "" {
			name = "Unspecified interface"
		}
		if now.Sub(t.interfaceNamesAt) <= t.interfaceNamesMaxAge && t.interfaceNames[id] != "" {
			name = t.interfaceNames[id]
		}
		s.labels["interface_name"] = name
	}
	if id := s.labels["proxy"]; hasProxy {
		name := id
		if now.Sub(t.policyNamesAt) <= policyMaxAge && t.policyNames[id] != "" {
			name = t.policyNames[id]
		}
		s.labels["proxy_name"] = name
	}
	return s
}
