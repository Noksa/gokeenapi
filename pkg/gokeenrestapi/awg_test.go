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
						Peer: []MockPeer{
							{
								Key:               "gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=",
								Endpoint:          "old-server.com:51820",
								KeepaliveInterval: 15,
								AllowedIPs:        []MockAllowedIP{{Address: "0.0.0.0", Mask: "0.0.0.0"}},
							},
						},
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

	createConfFile := func(content string) string {
		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "test.conf")
		Expect(os.WriteFile(confPath, []byte(content), 0644)).To(Succeed())
		return confPath
	}

	It("should update ASC and peer when both differ", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
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
PersistentKeepalive = 25`)

		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should fail for non-existent interface", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = abc
Address = 10.0.0.2/24
Jc = 1
Jmin = 2
Jmax = 3
S1 = 4
S2 = 5
H1 = 6
H2 = 7
H3 = 8
H4 = 9

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = x.com:51820
AllowedIPs = 0.0.0.0/0`)

		err := AwgConf.ConfigureOrUpdateInterface(confPath, "NonExistentInterface")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("doesn't have interface"))
	})

	It("should fail for empty path", func() {
		err := AwgConf.ConfigureOrUpdateInterface("", "Wireguard0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("conf-file flag is required"))
	})

	It("should work with plain WireGuard conf without ASC", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = new-server.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 30`)

		// No ASC in conf → only peer gets updated
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should update peer endpoint and keepalive", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
Jc = 40
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
Endpoint = new-endpoint.com:443
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 30`)

		// ASC matches, but peer endpoint+keepalive differ
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should handle full AWG 2.0 parameters", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
Jc = 40
Jmin = 5
Jmax = 95
S1 = 10
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4
S3 = 50
S4 = 60
I1 = 70
I2 = 80
I3 = 90
I4 = 100
I5 = 110

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = old-server.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 15`)

		// Peer unchanged, AWG 2.0 params are new → triggers ASC update
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should handle partial AWG 2.0 — only I1 specified", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
Jc = 40
Jmin = 5
Jmax = 95
S1 = 10
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4
I1 = 42

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = old-server.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 15`)

		// Only I1 present; S3, S4, I2-I5 default to "0" in the RCI command
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should handle partial AWG 2.0 — only S3 and S4 specified", func() {
		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
Jc = 40
Jmin = 5
Jmax = 95
S1 = 10
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4
S3 = 77
S4 = 88

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = old-server.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 15`)

		// S3+S4 present; I1-I5 default to "0"
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
	})

	It("should skip update when nothing changed", func() {
		// Set up mock with exact same values as conf file
		if server != nil {
			server.Close()
		}
		server = SetupMockRouterForTest(
			WithScInterfaces(map[string]MockScInterface{
				"Wireguard0": {
					Description: "Test WireGuard interface",
					IP:          MockIP{Address: "10.0.0.1/24"},
					Wireguard: MockWireguard{
						Asc: MockAsc{Jc: "50", Jmin: "5", Jmax: "95", S1: "10", S2: "20", H1: "1", H2: "2", H3: "3", H4: "4"},
						Peer: []MockPeer{
							{
								Key:               "gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=",
								Endpoint:          "example.com:51820",
								KeepaliveInterval: 25,
								AllowedIPs:        []MockAllowedIP{{Address: "0.0.0.0", Mask: "0.0.0.0"}},
							},
						},
					},
				},
			}),
		)

		confPath := createConfFile(`[Interface]
PrivateKey = cOFA+3p5IjkzIjkzIjkzIjkzIjkzIjkzIjkzIjkzIjk=
Address = 10.0.0.2/24
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
PersistentKeepalive = 25`)

		// Everything matches → no-op, no error
		Expect(AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")).To(Succeed())
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
		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "myawg.conf")
		Expect(os.WriteFile(confPath, []byte("[Interface]\nPrivateKey=abc\n"), 0644)).To(Succeed())

		// Expect error from network/post since no handler, but name fallback executed
		_, err := AwgConf.AddInterface(confPath, "")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("AWG DiffUpdate", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest(
			WithScInterfaces(map[string]MockScInterface{
				"Wireguard0": {
					Description: "Test WireGuard",
					IP:          MockIP{Address: "10.9.9.2/32"},
					Wireguard: MockWireguard{
						Asc: MockAsc{Jc: "3", Jmin: "41", Jmax: "115", S1: "110", S2: "52", H1: "100", H2: "200", H3: "300", H4: "400", S3: "52", S4: "19", I1: "56"},
						Peer: []MockPeer{
							{
								Key:               "ySQT/OAXa6htlgpxQLhFFjrVN/qlGoyEUNTAzYM9Fzs=",
								Endpoint:          "148.253.214.56:1235",
								KeepaliveInterval: 33,
								AllowedIPs:        []MockAllowedIP{{Address: "0.0.0.0", Mask: "0.0.0.0"}},
							},
						},
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

	createConf := func(content string) string {
		tmpDir := GinkgoT().TempDir()
		confPath := filepath.Join(tmpDir, "test.conf")
		Expect(os.WriteFile(confPath, []byte(content), 0644)).To(Succeed())
		return confPath
	}

	It("should return empty diff when nothing changed", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.9.9.2/32
Jc = 3
Jmin = 41
Jmax = 115
S1 = 110
S2 = 52
H1 = 100
H2 = 200
H3 = 300
H4 = 400
S3 = 52
S4 = 19
I1 = 56

[Peer]
PublicKey = ySQT/OAXa6htlgpxQLhFFjrVN/qlGoyEUNTAzYM9Fzs=
Endpoint = 148.253.214.56:1235
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 33`)

		diff, err := AwgConf.DiffUpdate(confPath, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(BeEmpty())
	})

	It("should show diff when endpoint changes", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.9.9.2/32
Jc = 3
Jmin = 41
Jmax = 115
S1 = 110
S2 = 52
H1 = 100
H2 = 200
H3 = 300
H4 = 400
S3 = 52
S4 = 19
I1 = 56

[Peer]
PublicKey = ySQT/OAXa6htlgpxQLhFFjrVN/qlGoyEUNTAzYM9Fzs=
Endpoint = 999.999.999.999:5555
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 33`)

		diff, err := AwgConf.DiffUpdate(confPath, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).NotTo(BeEmpty())
		Expect(diff).To(ContainSubstring("-Endpoint = 148.253.214.56:1235"))
		Expect(diff).To(ContainSubstring("+Endpoint = 999.999.999.999:5555"))
	})

	It("should show diff when ASC changes", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.9.9.2/32
Jc = 99
Jmin = 41
Jmax = 115
S1 = 110
S2 = 52
H1 = 100
H2 = 200
H3 = 300
H4 = 400

[Peer]
PublicKey = ySQT/OAXa6htlgpxQLhFFjrVN/qlGoyEUNTAzYM9Fzs=
Endpoint = 148.253.214.56:1235
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 33`)

		diff, err := AwgConf.DiffUpdate(confPath, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).NotTo(BeEmpty())
		Expect(diff).To(ContainSubstring("-Jc = 3"))
		Expect(diff).To(ContainSubstring("+Jc = 99"))
	})

	It("should not show address as changed when it matches", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.9.9.2/32
Jc = 3
Jmin = 41
Jmax = 115
S1 = 110
S2 = 52
H1 = 100
H2 = 200
H3 = 300
H4 = 400
S3 = 52
S4 = 19
I1 = 56

[Peer]
PublicKey = ySQT/OAXa6htlgpxQLhFFjrVN/qlGoyEUNTAzYM9Fzs=
Endpoint = 148.253.214.56:1235
AllowedIPs = 10.0.0.0/8, 192.168.0.0/16
PersistentKeepalive = 33`)

		diff, err := AwgConf.DiffUpdate(confPath, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).NotTo(ContainSubstring("-Address"))
		Expect(diff).NotTo(ContainSubstring("+Address"))
	})

	It("should fail for non-existent interface", func() {
		confPath := createConf(`[Interface]
PrivateKey = abc
Address = 10.0.0.1/24

[Peer]
PublicKey = key
Endpoint = x:1`)

		_, err := AwgConf.DiffUpdate(confPath, "NonExistent")
		Expect(err).To(HaveOccurred())
	})

	It("should fail for empty conf path", func() {
		_, err := AwgConf.DiffUpdate("", "Wireguard0")
		Expect(err).To(HaveOccurred())
	})
})
