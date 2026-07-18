package gokeenrestapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"

	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// domainServer starts a minimal HTTP server that serves domain lists.
// Each call to the returned setter changes the response body.
type domainServer struct {
	*httptest.Server
	body atomic.Value
}

func newDomainServer(initialBody string) *domainServer {
	ds := &domainServer{}
	ds.body.Store(initialBody)
	ds.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := ds.body.Load().(string)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	return ds
}

func (ds *domainServer) setBody(body string) { ds.body.Store(body) }

var _ = Describe("LoadDomainsFromURL", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "gokeenapi-url-test-*")
		Expect(err).NotTo(HaveOccurred())
		config.Cfg.DataDir = tmpDir
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		config.Cfg.DataDir = ""
	})

	It("should load domains from a successful HTTP response", func() {
		ds := newDomainServer("example.com\ntest.com\ngoogle.com\n")
		DeferCleanup(ds.Close)

		domains, err := DnsRouting.LoadDomainsFromURL(ds.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(ConsistOf("example.com", "test.com", "google.com"))
	})

	It("should skip comments and empty lines", func() {
		ds := newDomainServer("# comment\nexample.com\n\ntest.com\n# another comment\n")
		DeferCleanup(ds.Close)

		domains, err := DnsRouting.LoadDomainsFromURL(ds.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(ConsistOf("example.com", "test.com"))
	})

	It("should return error on non-200 response", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		DeferCleanup(srv.Close)

		_, err := DnsRouting.LoadDomainsFromURL(srv.URL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to fetch domain URL"))
	})

	It("should return error when server is unreachable", func() {
		_, err := DnsRouting.LoadDomainsFromURL("http://127.0.0.1:1") // refused
		Expect(err).To(HaveOccurred())
	})

	It("should return cached content on the second call", func() {
		var callCount atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			fmt.Fprint(w, "cached.com\n") //nolint:errcheck // test server, write error is irrelevant
		}))
		DeferCleanup(srv.Close)

		// First call — must hit the server
		domains1, err := DnsRouting.LoadDomainsFromURL(srv.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(domains1).To(ConsistOf("cached.com"))
		Expect(callCount.Load()).To(Equal(int32(1)))

		// Second call — must come from cache, no additional HTTP request
		domains2, err := DnsRouting.LoadDomainsFromURL(srv.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(domains2).To(ConsistOf("cached.com"))
		Expect(callCount.Load()).To(Equal(int32(1)), "second call should be served from cache")
	})

	It("should detect checksum change when content is updated after cache expiry", func() {
		ds := newDomainServer("original.com\n")
		DeferCleanup(ds.Close)

		// Prime the cache with a very short TTL
		Expect(gokeencache.SetURLContent(ds.URL, "original.com\n", 1*time.Millisecond)).To(Succeed())

		// Wait for TTL to expire so the cache is stale
		time.Sleep(5 * time.Millisecond)

		// Update the server content to trigger checksum-change detection
		ds.setBody("updated.com\nnew.com\n")

		domains, err := DnsRouting.LoadDomainsFromURL(ds.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(ConsistOf("updated.com", "new.com"))

		// Checksum should now reflect the new content
		newChecksum := gokeencache.GetURLChecksum(ds.URL)
		Expect(newChecksum).NotTo(BeEmpty())
	})

	It("should strip v2fly prefixes and validate domains", func() {
		ds := newDomainServer("full:example.com\ndomain:test.com\nnotvalid\ngoogle.com\n")
		DeferCleanup(ds.Close)

		domains, err := DnsRouting.LoadDomainsFromURL(ds.URL)
		Expect(err).NotTo(HaveOccurred())
		// "full:example.com" → "example.com", "domain:test.com" → "test.com",
		// "notvalid" skipped (no TLD), "google.com" kept
		Expect(domains).To(ConsistOf("example.com", "test.com", "google.com"))
	})
})
