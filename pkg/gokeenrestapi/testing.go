package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

// SetupMockServer creates a mock HTTP server for testing
func SetupMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// Auth endpoint
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("x-ndm-realm", "test-realm")
			w.Header().Set("x-ndm-challenge", "test-challenge")
			w.Header().Set("set-cookie", "session=test-session; Path=/")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	// Version endpoint
	mux.HandleFunc("/rci/show/version", func(w http.ResponseWriter, r *http.Request) {
		version := gokeenrestapimodels.Version{
			Model: "KN-1010",
			Title: "KeeneticOS 3.7.5",
		}
		encodeJSON(w, version)
	})

	// Interfaces endpoint
	mux.HandleFunc("/rci/show/interface", func(w http.ResponseWriter, r *http.Request) {
		interfaces := map[string]gokeenrestapimodels.RciShowInterface{
			"Wireguard0": {
				Id:          "Wireguard0",
				Type:        InterfaceTypeWireguard,
				Description: "Test WireGuard interface",
				Address:     "10.0.0.1/24",
				Connected:   StateConnected,
				Link:        StateUp,
				State:       StateUp,
			},
			"ISP": {
				Id:        "ISP",
				Type:      InterfaceTypePPPoE,
				Connected: StateConnected,
				Link:      StateUp,
				State:     StateUp,
			},
		}
		encodeJSON(w, interfaces)
	})

	// Single interface endpoint
	mux.HandleFunc("/rci/show/interface/", func(w http.ResponseWriter, r *http.Request) {
		interfaceId := r.URL.Path[len("/rci/show/interface/"):]
		if interfaceId == "Wireguard0" {
			iface := gokeenrestapimodels.RciShowInterface{
				Id:          "Wireguard0",
				Type:        InterfaceTypeWireguard,
				Description: "Test WireGuard interface",
				Address:     "10.0.0.1/24",
				Connected:   StateConnected,
				Link:        StateUp,
				State:       StateUp,
			}
			encodeJSON(w, iface)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	// SC interfaces endpoint
	mux.HandleFunc("/rci/show/sc/interface", func(w http.ResponseWriter, r *http.Request) {
		interfaces := map[string]gokeenrestapimodels.RciShowScInterface{
			"Wireguard0": {
				Description: "Test WireGuard interface",
			},
		}
		encodeJSON(w, interfaces)
	})

	// Hotspot endpoint
	mux.HandleFunc("/rci/show/ip/hotspot", func(w http.ResponseWriter, r *http.Request) {
		hotspot := gokeenrestapimodels.RciShowIpHotspot{
			Host: []gokeenrestapimodels.Host{
				{
					Name:     "test-device-1",
					Mac:      "aa:bb:cc:dd:ee:ff",
					IP:       "192.168.1.100",
					Hostname: "device1",
				},
				{
					Name:     "test-device-2",
					Mac:      "11:22:33:44:55:66",
					IP:       "192.168.1.101",
					Hostname: "device2",
				},
			},
		}
		encodeJSON(w, hotspot)
	})

	// Parse endpoint
	mux.HandleFunc("/rci/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var requests []gokeenrestapimodels.ParseRequest
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		responses := make([]gokeenrestapimodels.ParseResponse, len(requests))
		for i, req := range requests {
			responses[i] = gokeenrestapimodels.ParseResponse{
				Parse: gokeenrestapimodels.Parse{
					Status: []gokeenrestapimodels.Status{
						{Status: StatusOK, Code: "0", Message: fmt.Sprintf("Command executed: %s", req.Parse)},
					},
				},
			}
		}
		encodeJSON(w, responses)
	})

	// Running config endpoint
	mux.HandleFunc("/rci/show/running-config", func(w http.ResponseWriter, r *http.Request) {
		runningConfig := gokeenrestapimodels.RunningConfig{
			Message: []string{"test running config line 1", "test running config line 2"},
		}
		encodeJSON(w, runningConfig)
	})

	// System mode endpoint
	mux.HandleFunc("/rci/show/system/mode", func(w http.ResponseWriter, r *http.Request) {
		systemMode := gokeenrestapimodels.SystemMode{
			Active:   "router",
			Selected: "router",
		}
		encodeJSON(w, systemMode)
	})

	// IP route endpoints
	mux.HandleFunc("/rci/ip/route", func(w http.ResponseWriter, r *http.Request) {
		routes := []gokeenrestapimodels.RciIpRoute{
			{
				Network:   "192.168.1.0",
				Host:      "192.168.1.0",
				Mask:      "255.255.255.0",
				Interface: "Wireguard0",
				Auto:      false,
			},
		}
		encodeJSON(w, routes)
	})

	return httptest.NewServer(mux)
}

// SetupTestConfig configures the global config for testing
func SetupTestConfig(serverURL string) {
	config.Cfg = config.GokeenapiConfig{
		Keenetic: config.Keenetic{
			URL:      serverURL,
			Login:    "admin",
			Password: "password",
		},
	}
	// Reset client to use new config
	restyClient = nil
}

// Helper function to safely encode JSON responses in tests
func encodeJSON(w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
	}
}
