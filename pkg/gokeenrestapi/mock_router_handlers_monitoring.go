package gokeenrestapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// handleMonitoringBatch extends the unified mock with structured read-only RCI queries.
func (m *MockRouter) handleMonitoringBatch(w http.ResponseWriter, r *http.Request, commands []json.RawMessage) {
	if m.monitoringBatchDisabled {
		http.NotFound(w, r)
		return
	}
	responses := make([]any, len(commands))
	for i, command := range commands {
		var body map[string]json.RawMessage
		if json.Unmarshal(command, &body) != nil {
			http.Error(w, "invalid command", http.StatusBadRequest)
			return
		}
		responses[i] = m.monitoringTree(r, "/rci", body)
	}
	m.encodeJSON(w, responses)
}

func (m *MockRouter) monitoringTree(r *http.Request, path string, tree map[string]json.RawMessage) any {
	response := make(map[string]any)
	for key, raw := range tree {
		if key == "name" {
			continue
		}
		var children map[string]json.RawMessage
		if json.Unmarshal(raw, &children) != nil {
			continue
		}
		next := path + "/" + key
		if len(children) > 0 {
			if name, ok := children["name"]; ok {
				var value string
				if json.Unmarshal(name, &value) == nil {
					r = r.Clone(r.Context())
					q := r.URL.Query()
					q.Set("name", value)
					r.URL.RawQuery = q.Encode()
				}
			}
			response[key] = m.monitoringTree(r, next, children)
			continue
		}
		recorder := httptest.NewRecorder()
		request := r.Clone(r.Context())
		request.Method = http.MethodGet
		request.URL.Path = next
		m.monitoringServe(recorder, request)
		var value any
		if json.Unmarshal(recorder.Body.Bytes(), &value) != nil {
			value = map[string]any{"status": []any{map[string]string{"status": "error", "code": "unsupported"}}}
		}
		response[key] = value
	}
	return response
}

func (m *MockRouter) monitoringServe(w http.ResponseWriter, r *http.Request) {
	if m.monitoringOverride(w, r) {
		return
	}
	switch strings.TrimRight(r.URL.Path, "/") {
	case "/rci/show/version":
		m.handleVersion(w, r)
	case "/rci/show/interface":
		m.handleInterfaces(w, r)
	case "/rci/show/ip/hotspot":
		m.handleHotspot(w, r)
	default:
		http.NotFound(w, r)
	}
}

// MockMonitoringResponse controls one read-only endpoint in the unified mock.
type MockMonitoringResponse struct {
	Body       string
	StatusCode int
	Delay      time.Duration
}

// WithMonitoringResponses supplies firmware-specific read-only response fixtures.
func WithMonitoringResponses(responses map[string]MockMonitoringResponse) MockRouterOption {
	return func(m *MockRouter) { m.monitoringResponses = responses }
}

// WithMonitoringRequestObserver observes actual HTTP requests, not commands inside a batch.
func WithMonitoringRequestObserver(observer func(string, string)) MockRouterOption {
	return func(m *MockRouter) { m.monitoringObserver = observer }
}

// WithMonitoringAuth requires a router-specific authenticated cookie for RCI reads.
func WithMonitoringAuth(login, password, cookie string) MockRouterOption {
	return func(m *MockRouter) {
		m.monitoringLogin = login
		m.monitoringPassword = password
		m.sessionCookie = cookie
	}
}

// WithoutMonitoringBatch simulates firmware with only individual GET endpoints.
func WithoutMonitoringBatch() MockRouterOption {
	return func(m *MockRouter) { m.monitoringBatchDisabled = true }
}

func (m *MockRouter) monitoringHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.monitoringObserver != nil {
			m.monitoringObserver(r.Method, r.URL.Path)
		}
		m.mu.RLock()
		requireAuth := m.monitoringLogin != "" && r.URL.Path != "/auth"
		authorized := m.monitoringAuthenticated && r.Header.Get("Cookie") == m.sessionCookie
		m.mu.RUnlock()
		if requireAuth && !authorized {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet && m.monitoringOverride(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *MockRouter) monitoringOverride(w http.ResponseWriter, r *http.Request) bool {
	key := r.URL.Path
	if name := r.URL.Query().Get("name"); name != "" {
		key += "?name=" + name
	}
	fixture, ok := m.monitoringResponses[key]
	if !ok {
		return false
	}
	if fixture.Delay > 0 {
		select {
		case <-time.After(fixture.Delay):
		case <-r.Context().Done():
			return true
		}
	}
	status := fixture.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fixture.Body))
	return true
}
