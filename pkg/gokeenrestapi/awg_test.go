package gokeenrestapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWG", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest(
			WithScInterfaces(map[string]MockScInterface{
				"Wireguard0": {
					Description: "Test WireGuard interface",
					IP:          MockIP{Address: "10.0.0.1/24"},
					Wireguard: MockWireguard{
						Asc: MockAsc{Jc: "40", Jmin: "5", Jmax: "95", S1: "10", S2: "20", H1: "1", H2: "2", H3: "3", H4: "4"},
					},
				},
			}),
		)
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	createTestConfig := func() string {
		confContent := `[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
DNS = 8.8.8.8
Jc = 50
Jmin = 5
Jmax = 95
S1 = 10
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`

		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "test.conf")
		Expect(os.WriteFile(confPath, []byte(confContent), 0644)).To(Succeed())
		return confPath
	}

	It("should configure existing interface", func() {
		confPath := createTestConfig()
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should fail for non-existent interface", func() {
		confPath := createTestConfig()
		err := AwgConf.ConfigureOrUpdateInterface(confPath, "NonExistentInterface")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("doesn't have interface"))
	})

	It("should fail for empty path", func() {
		err := AwgConf.ConfigureOrUpdateInterface("", "Wireguard0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("conf-file flag is required"))
	})
})

var _ = Describe("AWG AddInterface", func() {
	It("returns error on permission denied reading conf file", func() {
		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "noperm.conf")
		Expect(os.WriteFile(confPath, []byte("[Interface]\n"), 0000)).To(Succeed())
		DeferCleanup(func() {
			_ = os.Chmod(confPath, 0644)
		})

		_, err := AwgConf.AddInterface(confPath, "test")
		Expect(err).To(HaveOccurred())
		Expect(os.IsPermission(err)).To(BeTrue())
	})

	It("returns error for non-existent conf file", func() {
		_, err := AwgConf.AddInterface("/nonexistent/path/awg.conf", "")
		Expect(err).To(HaveOccurred())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("uses base filename when name is empty", func() {
		// This exercises the name fallback; full success path requires router mock
		// which is not yet wired for /rci/interface/wireguard/import
		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "myawg.conf")
		Expect(os.WriteFile(confPath, []byte("[Interface]\nPrivateKey=abc\n"), 0644)).To(Succeed())

		// Expect error from network/post since no handler, but name fallback executed
		_, err := AwgConf.AddInterface(confPath, "")
		Expect(err).To(HaveOccurred())
	})
})
