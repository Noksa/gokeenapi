package gokeenrestapimodels

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
)

// MonitoringNumber accepts the numeric strings returned by some KeeneticOS commands.
// Valid distinguishes a missing/null field from an actual zero.
type MonitoringNumber struct {
	Value float64
	Valid bool
}

// UnmarshalJSON decodes a finite, nonnegative number or numeric string.
func (n *MonitoringNumber) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var value json.Number
	if json.Unmarshal(b, &value) != nil {
		return errors.New("invalid monitoring number")
	}
	parsed, err := strconv.ParseFloat(string(value), 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || parsed < 0 {
		return errors.New("invalid monitoring number")
	}
	n.Value, n.Valid = parsed, true
	return nil
}

// MonitoringSystem contains only system values needed by the exporter.
type MonitoringSystem struct {
	CPULoad          MonitoringNumber `json:"cpuload"`
	Uptime           MonitoringNumber `json:"uptime"`
	MemoryTotal      MonitoringNumber `json:"memtotal"`
	MemoryFree       MonitoringNumber `json:"memfree"`
	MemoryCache      MonitoringNumber `json:"memcache"`
	MemoryBuffers    MonitoringNumber `json:"membuffers"`
	ConnectionsTotal MonitoringNumber `json:"conntotal"`
	ConnectionsFree  MonitoringNumber `json:"connfree"`
}

// MonitoringDNS holds the per-policy DNS proxy statistics without configuration secrets.
type MonitoringDNS struct {
	Proxies []MonitoringDNSProxy `json:"proxy-status"`
}

// MonitoringDNSProxy contains a proxy name and ndnproxy statistics text.
type MonitoringDNSProxy struct {
	Name       string `json:"proxy-name"`
	Statistics string `json:"proxy-stat"`
}

// MonitoringHosts preserves optional activity and registration information.
type MonitoringHosts struct {
	Hosts []MonitoringHost `json:"host"`
}

// MonitoringHost contains only fields needed for aggregate device counts.
type MonitoringHost struct {
	Active     *bool  `json:"active"`
	Registered *bool  `json:"registered"`
	Link       string `json:"link"`
}

// MonitoringInterfaceStat is the read-only interface counter response.
type MonitoringInterfaceStat struct {
	RXBytes   MonitoringNumber `json:"rxbytes"`
	TXBytes   MonitoringNumber `json:"txbytes"`
	RXPackets MonitoringNumber `json:"rxpackets"`
	TXPackets MonitoringNumber `json:"txpackets"`
	RXErrors  MonitoringNumber `json:"rxerrors"`
	TXErrors  MonitoringNumber `json:"txerrors"`
	RXDropped MonitoringNumber `json:"rxdropped"`
	TXDropped MonitoringNumber `json:"txdropped"`
}
