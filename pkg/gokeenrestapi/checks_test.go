package gokeenrestapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckInterfaceId", func() {
	It("should succeed for non-empty id", func() {
		Expect(Checks.CheckInterfaceId("Wireguard0")).To(Succeed())
	})

	It("should fail for empty id", func() {
		err := Checks.CheckInterfaceId("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("please specify"))
	})
})

var _ = Describe("CheckInterfaceExists", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest()
	})

	AfterEach(func() {
		server.Close()
		CleanupTestConfig()
	})

	It("should succeed for an existing interface", func() {
		Expect(Checks.CheckInterfaceExists("Wireguard0")).To(Succeed())
	})

	It("should fail for a non-existent interface", func() {
		err := Checks.CheckInterfaceExists("NonExistent99")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("NonExistent99"))
	})
})

var _ = Describe("CheckComponentInstalled", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest()
	})

	AfterEach(func() {
		server.Close()
		CleanupTestConfig()
	})

	It("should return installed status for wireguard component", func() {
		installed, err := Checks.CheckComponentInstalled("wireguard")
		Expect(err).NotTo(HaveOccurred())
		Expect(installed).To(Equal("yes"))
	})

	It("should be case-insensitive", func() {
		installed, err := Checks.CheckComponentInstalled("WIREGUARD")
		Expect(err).NotTo(HaveOccurred())
		Expect(installed).To(Equal("yes"))
	})

	It("should return empty string for unknown component", func() {
		installed, err := Checks.CheckComponentInstalled("nonexistent-component")
		Expect(err).NotTo(HaveOccurred())
		Expect(installed).To(BeEmpty())
	})

	It("should return not-installed status for uninstalled component", func() {
		server.Close()
		CleanupTestConfig()
		server = SetupMockRouterForTest(WithComponents(map[string]gokeenrestapimodels.Component{
			"wireguard": {Group: "vpn", Installed: ""},
		}))

		installed, err := Checks.CheckComponentInstalled("wireguard")
		Expect(err).NotTo(HaveOccurred())
		Expect(installed).To(BeEmpty())
	})
})

var _ = Describe("CheckAWGInterfaceExistsFromConfFile", func() {
	var server *httptest.Server

	createConf := func(content string) string {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "test.conf")
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
		return path
	}

	BeforeEach(func() {
		server = SetupMockRouterForTest(
			WithScInterfaces(map[string]MockScInterface{
				"Wireguard0": {
					Description: "Existing VPN",
					// IP.Address stores the bare IP (no CIDR prefix) — matches how the router returns it
					IP: MockIP{Address: "10.0.0.1"},
					Wireguard: MockWireguard{
						Peer: []MockPeer{
							{
								Key:      "gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=",
								Endpoint: "vpn.example.com:51820",
							},
						},
					},
				},
			}),
		)
	})

	AfterEach(func() {
		server.Close()
		CleanupTestConfig()
	})

	It("should succeed when no matching interface exists", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.99.99.1/24

[Peer]
PublicKey = differentPublicKey1234567890123456789012345=
Endpoint = different.host.com:51820
AllowedIPs = 0.0.0.0/0`)

		Expect(Checks.CheckAWGInterfaceExistsFromConfFile(confPath)).To(Succeed())
	})

	It("should fail when a duplicate interface is found (same pubkey, endpoint, address)", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.0.0.1/24

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0`)

		err := Checks.CheckAWGInterfaceExistsFromConfFile(confPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
		Expect(err.Error()).To(ContainSubstring("Wireguard0"))
	})

	It("should succeed when only pubkey matches but endpoint differs", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.0.0.1/24

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = different.endpoint.com:12345
AllowedIPs = 0.0.0.0/0`)

		Expect(Checks.CheckAWGInterfaceExistsFromConfFile(confPath)).To(Succeed())
	})

	It("should fail for non-existent conf file", func() {
		err := Checks.CheckAWGInterfaceExistsFromConfFile("/nonexistent/path.conf")
		Expect(err).To(HaveOccurred())
	})

	It("should fail for conf file without Interface section", func() {
		confPath := createConf(`[Peer]
PublicKey = key
Endpoint = host:port`)

		err := Checks.CheckAWGInterfaceExistsFromConfFile(confPath)
		Expect(err).To(HaveOccurred())
	})
})
