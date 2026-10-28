package gokeenrestapi

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/noksa/gokeenapi/pkg/config"
)

// MonitoringQuery identifies one allowlisted, read-only RCI resource.
type MonitoringQuery struct {
	Path      string
	Interface string
}

// MonitoringResult contains one resource or its independent collection error.
type MonitoringResult struct {
	Data json.RawMessage
	Err  error
}

// MonitoringSession owns one router's HTTP connection and authentication state.
// Obtain sessions through Common.NewMonitoringSession, never through global Cfg.
type MonitoringSession struct {
	mu          sync.Mutex
	client      *resty.Client
	connection  config.Keenetic
	cookie      string
	requests    atomic.Uint64
	singleReads bool
}

// NewMonitoringSession creates a read-only session without network I/O.
func (*keeneticCommon) NewMonitoringSession(c config.Keenetic, timeout time.Duration) (*MonitoringSession, error) {
	if err := config.ValidateMonitoringConnection(c); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		return nil, errors.New("monitoring timeout must be positive")
	}
	if c.Timeout > 0 && c.Timeout < timeout {
		timeout = c.Timeout
	}
	client := resty.New().SetBaseURL(strings.TrimRight(c.URL, "/")).SetTimeout(timeout).SetCookieJar(nil).SetRedirectPolicy(resty.NoRedirectPolicy()).SetDisableWarn(true).SetResponseBodyLimit(8 << 20)
	if c.TLSSkipVerify {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	} //nolint:gosec // Explicit router-local configuration.
	return &MonitoringSession{client: client, connection: c}, nil
}

// Read collects allowlisted resources, batching operational show commands.
// Configuration resources always use GET: POSTing their CLI paths could mutate state.
func (s *MonitoringSession) Read(ctx context.Context, queries []MonitoringQuery) []MonitoringResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]MonitoringResult, len(queries))
	for _, q := range queries {
		if !validMonitoringQuery(q) {
			for i := range results {
				results[i].Err = errors.New("monitoring resource is not allowlisted")
			}
			return results
		}
	}
	if err := ctx.Err(); err != nil {
		for i := range results {
			results[i].Err = err
		}
		return results
	}
	var show []int
	for i, q := range queries {
		if strings.HasPrefix(q.Path, "/rci/show/") {
			show = append(show, i)
		}
	}
	for start := 0; start < len(show) && !s.singleReads; start += maxParseRequests {
		chunk := show[start:min(start+maxParseRequests, len(show))]
		bodies := make([]any, len(chunk))
		for i, index := range chunk {
			bodies[i] = monitoringBody(queries[index])
		}
		response, err := s.execute(ctx, http.MethodPost, "/rci/", bodies)
		if err == nil && (response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusMethodNotAllowed || response.StatusCode() == http.StatusNotImplemented) {
			s.singleReads = true
		} else {
			var rows []json.RawMessage
			if err == nil {
				err = monitoringHTTPError(response)
			}
			if err == nil {
				if json.Unmarshal(response.Body(), &rows) != nil || len(rows) != len(chunk) {
					err = errors.New("invalid RCI batch response length or format")
				}
			}
			for i, index := range chunk {
				results[index].Err = err
				if err == nil {
					results[index].Data, results[index].Err = extractMonitoringData(rows[i], queries[index])
				}
			}
			if err != nil {
				if monitoringFatalError(ctx, response, err) {
					failPendingMonitoringResults(results, err)
					return results
				}
				// A broken batch is local to show resources. Configuration GETs
				// remain independent, and repeating more batches would add load.
				for _, index := range show[start+len(chunk):] {
					results[index].Err = err
				}
				break
			}
		}
	}
	for i, q := range queries {
		if strings.HasPrefix(q.Path, "/rci/show/") && !s.singleReads {
			continue
		}
		request := s.client.R().SetContext(ctx)
		if q.Interface != "" {
			request.SetQueryParam("name", q.Interface)
		}
		response, err := s.executeRequest(request, http.MethodGet, q.Path, nil)
		if err == nil {
			err = monitoringHTTPError(response)
		}
		if err == nil {
			results[i].Data = response.Body()
			err = monitoringStatusError(results[i].Data)
		}
		results[i].Err = err
		if monitoringFatalError(ctx, response, err) {
			failPendingMonitoringResults(results, err)
			break
		}
	}
	return results
}

func monitoringFatalError(ctx context.Context, response *resty.Response, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil {
		return true
	}
	if errors.Is(err, resty.ErrResponseBodyTooLarge) {
		return false
	}
	return response == nil || response.StatusCode() == http.StatusUnauthorized
}

func failPendingMonitoringResults(results []MonitoringResult, err error) {
	for i := range results {
		if results[i].Data == nil && results[i].Err == nil {
			results[i].Err = err
		}
	}
}

func validMonitoringQuery(q MonitoringQuery) bool {
	switch q.Path {
	case "/rci/show/system", "/rci/show/dns-proxy", "/rci/show/interface", "/rci/show/ip/hotspot", "/rci/show/version", "/rci/ip/route", "/rci/object-group/fqdn", "/rci/dns-proxy/route":
		return q.Interface == ""
	case "/rci/show/interface/stat":
		return strings.TrimSpace(q.Interface) != ""
	default:
		return false
	}
}

func monitoringBody(q MonitoringQuery) any {
	if q.Interface != "" {
		return map[string]any{"show": map[string]any{"interface": map[string]any{"name": q.Interface, "stat": map[string]any{}}}}
	}
	var body any = map[string]any{}
	parts := strings.Split(strings.TrimPrefix(q.Path, "/rci/"), "/")
	for _, part := range slices.Backward(parts) {
		body = map[string]any{part: body}
	}
	return body
}

func extractMonitoringData(body json.RawMessage, q MonitoringQuery) (json.RawMessage, error) {
	for key := range strings.SplitSeq(strings.TrimPrefix(q.Path, "/rci/"), "/") {
		if err := monitoringStatusError(body); err != nil {
			return nil, err
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(body, &object) != nil {
			return nil, errors.New("invalid RCI resource response")
		}
		var ok bool
		body, ok = object[key]
		if !ok {
			return nil, errors.New("missing RCI resource in batch response")
		}
	}
	return body, monitoringStatusError(body)
}

func monitoringStatusError(body json.RawMessage) error {
	if !json.Valid(body) || string(body) == "null" {
		return errors.New("empty or invalid RCI response")
	}
	var object struct {
		Status []struct {
			Status string `json:"status"`
			Code   string `json:"code"`
		} `json:"status"`
	}
	if json.Unmarshal(body, &object) == nil {
		for _, status := range object.Status {
			if status.Status == StatusError {
				return fmt.Errorf("RCI command failed (code %s)", status.Code)
			}
		}
	}
	return nil
}

func monitoringHTTPError(response *resty.Response) error {
	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return fmt.Errorf("router returned HTTP %d", response.StatusCode())
	}
	return nil
}

func (s *MonitoringSession) execute(ctx context.Context, method, path string, body any) (*resty.Response, error) {
	return s.executeRequest(s.client.R().SetContext(ctx), method, path, body)
}

func (s *MonitoringSession) executeRequest(request *resty.Request, method, path string, body any) (*resty.Response, error) {
	response, err := s.send(request, method, path, body)
	if err != nil || response.StatusCode() != http.StatusUnauthorized {
		return response, err
	}
	if err := s.authenticate(request.Context()); err != nil {
		return nil, err
	}
	return s.send(request, method, path, body)
}

func (s *MonitoringSession) send(request *resty.Request, method, path string, body any) (*resty.Response, error) {
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	request.Header.Del("Cookie")
	if s.cookie != "" {
		request.SetHeader("Cookie", s.cookie)
	}
	if body != nil {
		request.SetBody(body)
	}
	s.requests.Add(1)
	response, err := request.Execute(method, path)
	if err != nil {
		return nil, fmt.Errorf("router request failed: %w", err)
	}
	return response, nil
}

func (s *MonitoringSession) authenticate(ctx context.Context) error {
	response, err := s.send(s.client.R().SetContext(ctx), http.MethodGet, "/auth", nil)
	if err != nil {
		return err
	}
	if response.StatusCode() == http.StatusOK {
		return nil
	}
	if response.StatusCode() != http.StatusUnauthorized {
		return monitoringHTTPError(response)
	}
	realm, challenge := response.Header().Get("x-ndm-realm"), response.Header().Get("x-ndm-challenge")
	cookie, _, _ := strings.Cut(response.Header().Get("Set-Cookie"), ";")
	if realm == "" || challenge == "" || !strings.Contains(cookie, "=") {
		return errors.New("incomplete router authentication challenge")
	}
	s.cookie = cookie
	response, err = s.send(s.client.R().SetContext(ctx), http.MethodPost, "/auth", map[string]string{"login": s.connection.Login, "password": monitoringPassword(s.connection.Login, s.connection.Password, realm, challenge)})
	if err != nil {
		return err
	}
	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		s.cookie = ""
		return fmt.Errorf("router authentication failed (HTTP %d)", response.StatusCode())
	}
	return nil
}

func monitoringPassword(login, password, realm, challenge string) string {
	md5Hash := md5.Sum([]byte(login + ":" + realm + ":" + password)) //nolint:gosec // Keenetic challenge-response protocol.
	shaHash := sha256.Sum256(fmt.Appendf(nil, "%s%x", challenge, md5Hash))
	return fmt.Sprintf("%x", shaHash)
}

// Requests returns the total number of HTTP attempts, including authentication.
func (s *MonitoringSession) Requests() uint64 { return s.requests.Load() }

// Close releases idle connections after polling has stopped.
func (s *MonitoringSession) Close() { s.client.GetClient().CloseIdleConnections() }
