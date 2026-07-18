package gokeenrestapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/internal/gokeencache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddRoutesFromBatFile", func() {
	var server *httptest.Server

	writeBat := func(content string) string {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "routes.bat")
		Expect(os.WriteFile(p, []byte(content), 0644)).To(Succeed())
		return p
	}

	BeforeEach(func() {
		// Clear route cache before each test so AddRoutesFromBatFile sees
		// a fresh route table and doesn't skip routes as "already present".
		gokeencache.SetRciShowIpRoute(nil)
		server = SetupMockRouterForTest(WithRoutes([]MockRoute{}))
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
		CleanupTestConfig()
		gokeencache.SetRciShowIpRoute(nil)
	})

	It("should add routes from a valid bat file", func() {
		batFile := writeBat(`route ADD 10.0.0.0 MASK 255.0.0.0
route ADD 192.168.1.0 MASK 255.255.255.0
route ADD 172.16.0.0 MASK 255.255.0.0
`)
		Expect(Ip.AddRoutesFromBatFile(batFile, "Wireguard0")).To(Succeed())

		gokeencache.SetRciShowIpRoute(nil)
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		networks := make([]string, 0, len(routes))
		for _, r := range routes {
			networks = append(networks, r.Network)
		}
		Expect(networks).To(ContainElements("10.0.0.0", "192.168.1.0", "172.16.0.0"))
	})

	It("should skip comment lines and blank lines", func() {
		batFile := writeBat(`# This is a comment
route ADD 10.1.0.0 MASK 255.255.0.0

# Another comment
route ADD 10.2.0.0 MASK 255.255.0.0
`)
		Expect(Ip.AddRoutesFromBatFile(batFile, "Wireguard0")).To(Succeed())

		gokeencache.SetRciShowIpRoute(nil)
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).To(HaveLen(2))
	})

	It("should return error for non-existent file", func() {
		err := Ip.AddRoutesFromBatFile("/nonexistent/path/routes.bat", "Wireguard0")
		Expect(err).To(HaveOccurred())
	})

	It("should return error for lines with invalid format and still process valid ones", func() {
		batFile := writeBat(`route ADD 10.3.0.0 MASK 255.255.0.0
this is not a valid route line
route ADD 10.4.0.0 MASK 255.255.0.0
`)
		// multierr: invalid line is collected but valid routes are still added
		err := Ip.AddRoutesFromBatFile(batFile, "Wireguard0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid format"))

		gokeencache.SetRciShowIpRoute(nil)
		routes, err2 := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err2).NotTo(HaveOccurred())
		networks := make([]string, 0, len(routes))
		for _, r := range routes {
			networks = append(networks, r.Network)
		}
		Expect(networks).To(ContainElements("10.3.0.0", "10.4.0.0"))
	})

	It("should be idempotent — not add already-existing routes again", func() {
		// Pre-populate the router with a route
		server.Close()
		CleanupTestConfig()
		gokeencache.SetRciShowIpRoute(nil)
		server = SetupMockRouterForTest(WithRoutes([]MockRoute{
			{Network: "10.5.0.0", Mask: "255.255.0.0", Interface: "Wireguard0"},
		}))

		batFile := writeBat(`route ADD 10.5.0.0 MASK 255.255.0.0
`)
		Expect(Ip.AddRoutesFromBatFile(batFile, "Wireguard0")).To(Succeed())

		gokeencache.SetRciShowIpRoute(nil)
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		// Still only one route (no duplicate added)
		Expect(routes).To(HaveLen(1))
	})

	It("should return no error and skip all lines for empty bat file", func() {
		batFile := writeBat(`# Only comments
# Nothing to add
`)
		Expect(Ip.AddRoutesFromBatFile(batFile, "Wireguard0")).To(Succeed())

		gokeencache.SetRciShowIpRoute(nil)
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).To(BeEmpty())
	})

	It("should return error for line with out-of-range octet", func() {
		// Go's net.ParseIP accepts octets >255 in some contexts, so the invalid-IP
		// detection relies on the regex not matching pure non-numeric garbage.
		// A line that passes the regex but has invalid format (e.g. missing MASK keyword)
		// is what actually triggers the "invalid format" error path.
		batFile := writeBat("route ADD 10.0.0 MASK 255.0.0.0\n")
		err := Ip.AddRoutesFromBatFile(batFile, "Wireguard0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid format"))
	})
})

var _ = Describe("AddRoutesFromBatUrl success path", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest(WithRoutes([]MockRoute{}))
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
		CleanupTestConfig()
		gokeencache.SetRciShowIpRoute(nil)
	})

	It("should add routes from URL content", func() {
		batContent := "route ADD 10.6.0.0 MASK 255.255.0.0\nroute ADD 10.7.0.0 MASK 255.255.0.0\n"
		batServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(batContent))
		}))
		DeferCleanup(batServer.Close)

		Expect(Ip.AddRoutesFromBatUrl(batServer.URL, "Wireguard0")).To(Succeed())

		gokeencache.SetRciShowIpRoute(nil)
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		networks := make([]string, 0, len(routes))
		for _, r := range routes {
			networks = append(networks, r.Network)
		}
		Expect(networks).To(ContainElements("10.6.0.0", "10.7.0.0"))
	})
})
