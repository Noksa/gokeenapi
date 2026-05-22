package gokeenrestapi

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"go.uber.org/multierr"
)

const (
	cacheCleanupPeriod = time.Hour * 24 * 7
	maxParseRequests   = 100
	defaultTimeout     = time.Second * 30
)

var (
	restyClient     *resty.Client
	restyClientOnce sync.Once

	cacheCleanMu    sync.Mutex
	cleanedOldCache bool

	// Common provides core API functionality for authentication and router communication
	Common keeneticCommon
)

type keeneticCommon struct {
}

type keeneticCacheFile struct {
	Cookie keeneticCacheCookie `json:"cookie"`
	path   string
}
type keeneticCacheCookie struct {
	Value      string    `json:"value"`
	UpdateTime time.Time `json:"update_time"`
}

func (f *keeneticCacheFile) Save() error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(f.path, b, 0600)
	return err
}

func (c *keeneticCommon) getKeeneticCacheFile() (keeneticCacheFile, error) {
	gokeenDir, err := gokeencache.GetGokeenDir()
	if err != nil {
		return keeneticCacheFile{}, err
	}

	cacheCleanMu.Lock()
	needClean := !cleanedOldCache
	if needClean {
		cleanedOldCache = true
	}
	cacheCleanMu.Unlock()
	if needClean {
		err = filepath.WalkDir(gokeenDir, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if time.Since(info.ModTime()) >= cacheCleanupPeriod {
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			cacheCleanMu.Lock()
			cleanedOldCache = false // allow retry on next call
			cacheCleanMu.Unlock()
			return keeneticCacheFile{}, err
		}
	}
	bHash := sha256.Sum256(fmt.Appendf(nil, "%v-%v", config.Cfg.Keenetic.URL, config.Cfg.Keenetic.Login))
	hashString := fmt.Sprintf("%x", bHash)
	keeeticFile := path.Join(gokeenDir, fmt.Sprintf("%v.json", hashString))
	_, statErr := os.Stat(keeeticFile)
	if statErr != nil {
		if !errors.Is(statErr, os.ErrNotExist) {
			return keeneticCacheFile{}, statErr
		}
		err = os.WriteFile(keeeticFile, []byte("{}"), 0600)
		if err != nil {
			return keeneticCacheFile{}, err
		}
	}
	var keeneticCache keeneticCacheFile
	b, err := os.ReadFile(keeeticFile)
	if err != nil {
		return keeneticCacheFile{}, err
	}
	err = json.Unmarshal(b, &keeneticCache)
	keeneticCache.path = keeeticFile
	return keeneticCache, err
}

func (c *keeneticCommon) getAuthCookie() (string, error) {
	cache, err := c.getKeeneticCacheFile()
	if err != nil {
		return "", err
	}
	return cache.Cookie.Value, nil
}

func (c *keeneticCommon) writeAuthCookie(cookie string) error {
	cache, err := c.getKeeneticCacheFile()
	if err != nil {
		return err
	}
	cache.Cookie.Value = cookie
	cache.Cookie.UpdateTime = time.Now()
	return cache.Save()
}

// performAuth handles the actual authentication process with a specific client
func (c *keeneticCommon) performAuth(client *resty.Client) error {
	response, err := client.R().Get("/auth")
	if err != nil {
		return fmt.Errorf("failed to connect to router: %w", err)
	}

	if response == nil {
		return errors.New("no response from router")
	}

	switch response.StatusCode() {
	case http.StatusUnauthorized:
		realm := response.Header().Get("x-ndm-realm")
		token := response.Header().Get("x-ndm-challenge")
		setCookieStr := response.Header().Get("set-cookie")
		setCookieStrSplitted := strings.Split(setCookieStr, ";")
		cookieToSet := setCookieStrSplitted[0]

		if err := c.writeAuthCookie(cookieToSet); err != nil {
			return fmt.Errorf("failed to save auth cookie: %w", err)
		}

		secondRequest := client.R()
		md5Hash := md5.New()
		if _, err := fmt.Fprintf(md5Hash, "%v:%v:%v", config.Cfg.Keenetic.Login, realm, config.Cfg.Keenetic.Password); err != nil {
			return fmt.Errorf("failed to create MD5 hash: %w", err)
		}
		md5HashArg := md5Hash.Sum(nil)
		md5HashStr := hex.EncodeToString(md5HashArg)

		sha256Hash := sha256.New()
		if _, err := fmt.Fprintf(sha256Hash, "%v%v", token, md5HashStr); err != nil {
			return fmt.Errorf("failed to create SHA256 hash: %w", err)
		}
		sha256HashArg := sha256Hash.Sum(nil)
		sha256HashStr := hex.EncodeToString(sha256HashArg)

		secondRequest.SetBody(struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}{
			Login:    config.Cfg.Keenetic.Login,
			Password: sha256HashStr,
		})

		// set cookie globally
		client.Header.Set("Cookie", cookieToSet)
		secondRequest.Header.Set("Cookie", cookieToSet)

		response, err = secondRequest.Post("/auth")
		if err != nil {
			return fmt.Errorf("authentication request failed: %w", err)
		}

		if response.StatusCode() == http.StatusUnauthorized {
			return errors.New("authentication failed: verify your login and password")
		}

	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil

	default:
		return fmt.Errorf("router unavailable (status: %d %s)", response.StatusCode(), response.Status())
	}

	return nil
}

// Ping checks if the router is reachable by attempting a simple GET request
// This is faster than waiting for authentication to timeout
func (c *keeneticCommon) Ping() error {
	client := resty.New()
	client.SetDisableWarn(true)
	client.SetCookieJar(nil)
	client.SetTimeout(time.Second * 5) // Short timeout for ping
	client.SetBaseURL(config.Cfg.Keenetic.URL)
	if config.Cfg.Keenetic.TLSSkipVerify {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	}

	response, err := client.R().Get("/rci/show/version")
	if err != nil {
		return fmt.Errorf("router is not reachable: %w", err)
	}

	if response == nil {
		return errors.New("router is not reachable: no response")
	}

	// We expect either 200 (if no auth required) or 401 (auth required)
	// Both mean the router is reachable
	if response.StatusCode() != http.StatusOK && response.StatusCode() != http.StatusUnauthorized {
		return fmt.Errorf("router returned unexpected status: %d %s", response.StatusCode(), response.Status())
	}

	return nil
}

// Auth authenticates with the Keenetic router using configured credentials
// Handles the router's challenge-response authentication mechanism and caches the session
func (c *keeneticCommon) Auth() error {
	// First check if router is reachable
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Checking connectivity to %v", color.CyanString("Keenetic")), func() error {
		return c.Ping()
	})
	if err != nil {
		return err
	}

	var version gokeenrestapimodels.Version

	err = gokeenspinner.WrapWithSpinnerAndOptions(fmt.Sprintf("Authorizing in %v", color.CyanString("Keenetic")), func(opts *gokeenspinner.SpinnerOptions) error {
		client, err := c.GetApiClient()
		if err != nil {
			return err
		}
		if err := c.performAuth(client); err != nil {
			return err
		}
		if _, _, err := c.CheckRouterMode(); err != nil {
			return err
		}
		var vErr error
		version, vErr = c.Version()
		if vErr != nil {
			return vErr
		}

		opts.AddActionAfterSpinner(func() {
			gokeenlog.InfoSubStepf("%v: %v", color.BlueString("Router"), color.CyanString(version.Model))
			gokeenlog.InfoSubStepf("%v: %v", color.BlueString("OS version"), color.CyanString(version.Title))
		})

		return nil
	})
	if err != nil {
		return err
	}

	gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
		runtime.RouterInfo.Version = version
	})
	return nil
}

// Version retrieves the router's version information including model and OS version
func (c *keeneticCommon) Version() (gokeenrestapimodels.Version, error) {
	b, err := c.ExecuteGetSubPath("/rci/show/version")
	if err != nil {
		return gokeenrestapimodels.Version{}, err
	}
	var version gokeenrestapimodels.Version
	err = json.Unmarshal(b, &version)
	return version, err
}

// ExecutePostParse executes one or more CLI commands on the router via RCI interface
// Automatically batches commands in groups of 50 for optimal performance
func (c *keeneticCommon) ExecutePostParse(parse ...gokeenrestapimodels.ParseRequest) ([]gokeenrestapimodels.ParseResponse, error) {
	parseCopy := parse
	var parseResponses []gokeenrestapimodels.ParseResponse
	var mErr error
	for len(parseCopy) > 0 {
		apiClient, err := c.GetApiClient()
		if err != nil {
			return parseResponses, err
		}
		request := apiClient.R()
		maxParse := maxParseRequests
		currentLen := len(parseCopy)
		if currentLen < maxParse {
			maxParse = currentLen
		}
		var parseRequest []gokeenrestapimodels.ParseRequest
		for i := 0; i < maxParse; i++ {
			parseRequest = append(parseRequest, parseCopy[i])
		}
		parseCopy = parseCopy[maxParse:]
		request.SetBody(parseRequest)
		response, err := request.Post("/rci/")
		if response != nil {
			if response.StatusCode() != http.StatusOK {
				mErr = multierr.Append(mErr, fmt.Errorf("wrong status code in response from api: %s", response.Status()))
			}
			var parseResponse []gokeenrestapimodels.ParseResponse
			decodeErr := json.Unmarshal(response.Body(), &parseResponse)
			mErr = multierr.Append(mErr, decodeErr)
			for i, myParse := range parseResponse {
				if i == 0 {
					parseResponse[i].Parse.DynamicData = string(response.Body())
				}
				for _, status := range myParse.Parse.Status {
					if status.Status == StatusError {
						mErr = multierr.Append(mErr, fmt.Errorf("%s - %s - %s - %s", status.Status, status.Code, status.Ident, status.Message))
					}
				}
			}
			parseResponses = append(parseResponses, parseResponse...)
		}
		mErr = multierr.Append(mErr, err)
	}
	return parseResponses, mErr
}

// ExecuteGetSubPath performs a GET request to the specified API endpoint
func (c *keeneticCommon) ExecuteGetSubPath(path string) ([]byte, error) {
	apiClient, err := c.GetApiClient()
	if err != nil {
		return nil, err
	}
	response, err := apiClient.R().Get(path)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Body(), nil
	}
	return []byte{}, errors.New("no response from keenetic api")
}

// ExecutePostSubPath performs a POST request to the specified API endpoint with a request body
func (c *keeneticCommon) ExecutePostSubPath(path string, body any) ([]byte, error) {
	apiClient, err := c.GetApiClient()
	if err != nil {
		return nil, err
	}
	response, err := apiClient.R().SetBody(body).Post(path)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Body(), nil
	}
	return []byte{}, errors.New("no response from keenetic api")
}

// authRetried is the context key used to prevent infinite retry loops in authRetryMiddleware.
type authRetriedKey struct{}

// authRetryMiddleware handles 401 responses by re-authenticating and retrying the request.
// It uses a context flag to ensure the retry is performed at most once per request.
func (c *keeneticCommon) authRetryMiddleware(client *resty.Client, resp *resty.Response) error {
	if resp.StatusCode() == http.StatusUnauthorized && resp.Request.RawRequest.URL.Path != "/auth" {
		// Prevent infinite loop: if this request is already a retry, do not retry again.
		if resp.Request.Context().Value(authRetriedKey{}) != nil {
			return nil
		}

		// Clear the current cookie and perform direct authentication
		client.Header.Del("Cookie")

		if err := c.performAuth(client); err != nil {
			return err
		}

		// Retry the original request with new authentication, marking it as already retried.
		retryReq := resp.Request.SetContext(context.WithValue(resp.Request.Context(), authRetriedKey{}, true))
		retryReq.Header.Del("Cookie")
		retryReq.Header.Set("Cookie", client.Header.Get("Cookie"))

		retryResp, err := retryReq.Execute(resp.Request.Method, resp.Request.URL)
		if err != nil {
			return err
		}

		// Replace the response content with the retry response
		*resp = *retryResp
	}
	return nil
}

// GetApiClient returns a configured HTTP client for API requests with authentication
func (c *keeneticCommon) GetApiClient() (*resty.Client, error) {
	restyClientOnce.Do(func() {
		restyClient = resty.New()
		restyClient.SetDisableWarn(true)
		restyClient.SetCookieJar(nil)
		restyClient.SetTimeout(defaultTimeout)
		restyClient.OnAfterResponse(c.authRetryMiddleware)
		restyClient.RetryCount = 3
	})
	// do it each time to have clean client
	restyClient.SetBaseURL(config.Cfg.Keenetic.URL)
	if config.Cfg.Keenetic.TLSSkipVerify {
		restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	} else {
		restyClient.SetTLSClientConfig(&tls.Config{}) //nolint:gosec
	}
	if restyClient.Header.Get("Cookie") == "" {
		cookie, err := c.getAuthCookie()
		if err != nil {
			return nil, err
		}
		if cookie != "" {
			restyClient.Header.Set("Cookie", cookie)
		}
	}
	return restyClient, nil
}

// ShowRunningConfig retrieves the current running configuration from the router
func (c *keeneticCommon) ShowRunningConfig() (gokeenrestapimodels.RunningConfig, error) {
	var runningConfig gokeenrestapimodels.RunningConfig
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Fetching %v", color.CyanString("running-config")), func() error {
		b, err := c.ExecuteGetSubPath("/rci/show/running-config")
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &runningConfig)
		return err
	})
	return runningConfig, err
}

// SaveConfigParseRequest returns a parse request to save the current configuration
func (c *keeneticCommon) SaveConfigParseRequest() gokeenrestapimodels.ParseRequest {
	return gokeenrestapimodels.ParseRequest{Parse: "system configuration save"}
}

// EnsureSaveConfigAtEnd ensures SaveConfigParseRequest is at the end of parseSlice exactly once.
// Any existing occurrences of the save command are removed before appending it at the end.
func (c *keeneticCommon) EnsureSaveConfigAtEnd(parseSlice []gokeenrestapimodels.ParseRequest) []gokeenrestapimodels.ParseRequest {
	saveConfig := c.SaveConfigParseRequest()

	filtered := parseSlice[:0]
	for _, req := range parseSlice {
		if req.Parse != saveConfig.Parse {
			filtered = append(filtered, req)
		}
	}

	return append(filtered, saveConfig)
}

func (c *keeneticCommon) SaveConfig() error {
	parseRequest := c.SaveConfigParseRequest()
	_, err := c.ExecutePostParse(parseRequest)
	return err
}

// CheckRouterMode verifies that the router is in router mode (not extender mode)
func (c *keeneticCommon) CheckRouterMode() (string, string, error) {
	b, err := c.ExecuteGetSubPath("/rci/show/system/mode")
	if err != nil {
		return "", "", err
	}
	var systemMode gokeenrestapimodels.SystemMode
	err = json.Unmarshal(b, &systemMode)
	if err != nil {
		return "", "", err
	}
	if systemMode.Active != "router" || systemMode.Selected != "router" {
		return systemMode.Active, systemMode.Selected, fmt.Errorf("router is not in router mode (active: %s, selected: %s). Only router mode is supported", systemMode.Active, systemMode.Selected)
	}
	return systemMode.Active, systemMode.Selected, nil
}
