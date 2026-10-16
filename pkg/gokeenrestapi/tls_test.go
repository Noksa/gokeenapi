package gokeenrestapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TLS skip-verify", func() {
	var tlsServer *httptest.Server

	BeforeEach(func() {
		// Other specs share package-level clients; random order must not retain
		// a client configured for a router that has already been shut down.
		restyClient = nil
		restyClientOnce = sync.Once{}
		urlClient = nil
		urlClientOnce = sync.Once{}
		// Create an HTTPS test server with a self-signed certificate.
		// The /rci/show/version endpoint is the one Ping calls first.
		tlsServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/rci/show/version":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"model": "KN-test", "title": "1.0"})
			case "/rci/show/failure":
				w.WriteHeader(http.StatusServiceUnavailable)
			case "/auth":
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			default:
				w.WriteHeader(http.StatusOK)
			}
		}))
	})

	AfterEach(func() {
		tlsServer.Close()
		CleanupTestConfig()
		restyClient = nil
		restyClientOnce = sync.Once{}
		urlClient = nil
		urlClientOnce = sync.Once{}
	})

	Context("Ping", func() {
		It("should succeed when tls_skip_verify is true", func() {
			tmpDir := GinkgoT().TempDir()
			config.Cfg = config.GokeenapiConfig{
				Keenetic: config.Keenetic{
					URL:           tlsServer.URL,
					Login:         "admin",
					Password:      "pass",
					TLSSkipVerify: true,
				},
				DataDir: tmpDir,
			}

			Expect(Common.Ping()).To(Succeed())
		})

		It("should fail when tls_skip_verify is false (default)", func() {
			tmpDir := GinkgoT().TempDir()
			config.Cfg = config.GokeenapiConfig{
				Keenetic: config.Keenetic{
					URL:           tlsServer.URL,
					Login:         "admin",
					Password:      "pass",
					TLSSkipVerify: false,
				},
				DataDir: tmpDir,
			}

			err := Common.Ping()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("router is not reachable"))
		})
	})

	Context("GetApiClient", func() {
		It("should produce a client that connects when tls_skip_verify is true", func() {
			tmpDir := GinkgoT().TempDir()
			config.Cfg = config.GokeenapiConfig{
				Keenetic: config.Keenetic{
					URL:           tlsServer.URL,
					Login:         "admin",
					Password:      "pass",
					TLSSkipVerify: true,
				},
				DataDir: tmpDir,
			}
			restyClient = nil
			restyClientOnce = sync.Once{}

			client, err := Common.GetApiClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			resp, err := client.R().Get("/rci/show/version")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
		})

		It("should produce a client that fails TLS when tls_skip_verify is false", func() {
			tmpDir := GinkgoT().TempDir()
			config.Cfg = config.GokeenapiConfig{
				Keenetic: config.Keenetic{
					URL:           tlsServer.URL,
					Login:         "admin",
					Password:      "pass",
					TLSSkipVerify: false,
				},
				DataDir: tmpDir,
			}
			restyClient = nil
			restyClientOnce = sync.Once{}

			client, err := Common.GetApiClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			_, err = client.R().Get("/rci/show/version")
			Expect(err).To(HaveOccurred())
		})
	})

	It("returns a router GET HTTP failure as an error", func() {
		config.Cfg = config.GokeenapiConfig{Keenetic: config.Keenetic{
			URL: tlsServer.URL, TLSSkipVerify: true,
		}, DataDir: GinkgoT().TempDir()}
		body, err := Common.ExecuteGetSubPath("/rci/show/failure")
		Expect(err).To(MatchError(ContainSubstring("HTTP 503")))
		Expect(body).To(BeNil())
	})

	It("keeps external URL certificate verification enabled when router TLS verification is disabled", func() {
		config.Cfg = config.GokeenapiConfig{Keenetic: config.Keenetic{
			URL: tlsServer.URL, TLSSkipVerify: true,
		}, DataDir: GinkgoT().TempDir()}
		_, err := GetURLClient().R().Get(tlsServer.URL)
		Expect(err).To(HaveOccurred())
	})
})
