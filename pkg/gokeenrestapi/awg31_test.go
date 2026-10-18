package gokeenrestapi

import (
	"encoding/base64"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWG 3.1", func() {
	var server *httptest.Server

	createConf := func(contents string) string {
		path := filepath.Join(GinkgoT().TempDir(), "awg.conf")
		Expect(os.WriteFile(path, []byte(contents), 0600)).To(Succeed())
		return path
	}
	key := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	conf := func(extra string) string {
		return `[Interface]
PrivateKey = ignored-by-update
Address = 10.0.0.2/24
Jc = 40
Jmin = 5
Jmax = 95
S1 = 12
S2 = 20
H1 = 1000000-1100000
H2 = 2
H3 = 3
H4 = 4
S3 = 50
S4 = 60
I1 = 70
HeaderProtectionKey = ` + key + `
ContentPaddingAddition = 1-2
RekeyAfterTime = 3-4
RekeyTimeout = 5-6
RejectAfterTime = 7-8
KeepaliveTimeout = 9-10
MaxHandshakeAttempts = 11-12
RandomTrailers = on
DisableCookies = off
` + extra + `
[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25-35
`
	}

	BeforeEach(func() {
		server = SetupMockRouterForTest(WithVersion("5.2"))
	})
	AfterEach(func() {
		server.Close()
		CleanupTestConfig()
	})

	It("parses every 3.1 field and resolves keepalive range", func() {
		asc, peer, err := parseConfFile(createConf(conf("")))
		Expect(err).NotTo(HaveOccurred())
		Expect(asc.hasAWG3()).To(BeTrue())
		Expect(asc.AWG3.HeaderProtectionKey).To(Equal(key))
		Expect(asc.AWG3.ContentPaddingAddition).To(Equal(awgRange{1, 2}))
		Expect(asc.AWG3.RandomTrailers).To(BeTrue())
		Expect(asc.AWG3.DisableCookies).To(BeFalse())
		Expect(peer.PersistentKeepalive).To(Equal(25))
		payload, err := asc.awg3Payload()
		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(HaveLen(31))
		Expect(payload["h1"]).To(Equal("1000000-1100000"))
		Expect(payload["random-trailers"]).To(Equal(1))
	})

	It("recognizes an explicitly disabled 3.1 option", func() {
		asc, _, err := parseConfFile(createConf(strings.Replace(conf(""), "HeaderProtectionKey = "+key, "", 1)))
		Expect(err).NotTo(HaveOccurred())
		Expect(asc.hasAWG3()).To(BeTrue())
	})

	It("rejects malformed 3.1 fields before updating", func() {
		for _, replacement := range []string{
			"ContentPaddingAddition = 3-2",
			"ContentPaddingAddition = 1-70000",
			"RandomTrailers = maybe",
			"HeaderProtectionKey = bad-key",
			"PersistentKeepalive = nope",
		} {
			broken := conf("")
			switch {
			case strings.HasPrefix(replacement, "ContentPaddingAddition"):
				broken = strings.Replace(broken, "ContentPaddingAddition = 1-2", replacement, 1)
			case strings.HasPrefix(replacement, "RandomTrailers"):
				broken = strings.Replace(broken, "RandomTrailers = on", replacement, 1)
			case strings.HasPrefix(replacement, "HeaderProtectionKey"):
				broken = strings.Replace(broken, "HeaderProtectionKey = "+key, replacement, 1)
			default:
				broken = strings.Replace(broken, "PersistentKeepalive = 25-35", replacement, 1)
			}
			Expect(AwgConf.ConfigureOrUpdateInterface(createConf(broken), "Wireguard0")).To(HaveOccurred())
		}
	})

	It("rejects a malformed 3.1 import even without a header key", func() {
		broken := strings.Replace(conf(""), "HeaderProtectionKey = "+key, "", 1)
		broken = strings.Replace(broken, "ContentPaddingAddition = 1-2", "ContentPaddingAddition = 9-2", 1)
		_, err := AwgConf.AddInterface(createConf(broken), "Broken")
		Expect(err).To(MatchError(ContainSubstring("ContentPaddingAddition")))
	})

	It("rejects header protection without enough S padding", func() {
		broken := strings.Replace(conf(""), "S4 = 60", "S4 = 11", 1)
		_, err := AwgConf.AddInterface(createConf(broken), "Broken")
		Expect(err).To(MatchError(ContainSubstring("S4 must be at least 12")))
	})

	It("keeps the legacy ASC path for a keepalive-only range", func() {
		legacy := `[Interface]
Jc = 4
Jmin = 10
Jmax = 50
S1 = 50
S2 = 76
H1 = 1
H2 = 2
H3 = 3
H4 = 4
[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
PersistentKeepalive = 25-35
`
		path := createConf(legacy)
		asc, peer, err := parseConfFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(asc.hasAWG3()).To(BeFalse())
		Expect(peer.PersistentKeepalive).To(Equal(25))
		created, err := AwgConf.AddInterface(path, "LegacyRange")
		Expect(err).NotTo(HaveOccurred())
		Expect(AwgConf.ConfigureOrUpdateInterface(path, created.Created)).To(Succeed())
	})

	It("applies a complete 3.1 block and becomes idempotent", func() {
		path := createConf(conf(""))
		Expect(AwgConf.ConfigureOrUpdateInterface(path, "Wireguard0")).To(Succeed())
		diff, err := AwgConf.DiffUpdate(path, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(BeEmpty())
		Expect(AwgConf.ConfigureOrUpdateInterface(path, "Wireguard0")).To(Succeed())
	})

	It("imports a 3.1 config with range keepalive and then applies it", func() {
		path := createConf(conf(""))
		created, err := AwgConf.AddInterface(path, "NewAWG")
		Expect(err).NotTo(HaveOccurred())
		Expect(created.Created).To(Equal("Wireguard1"))
		Expect(AwgConf.ConfigureOrUpdateInterface(path, created.Created)).To(Succeed())
		state, err := Interface.GetInterfaceViaRciShowScInterfaces(created.Created)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Wireguard.Peer).To(HaveLen(1))
		Expect(state.Wireguard.Peer[0].KeepaliveInterval.Interval).To(Equal(25))
	})

	It("redacts the header key in a dry-run diff", func() {
		diff, err := AwgConf.DiffUpdate(createConf(conf("")), "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(ContainSubstring("HeaderProtectionKey"))
		Expect(diff).NotTo(ContainSubstring(key))
	})

	It("rejects unsupported firmware", func() {
		server.Close()
		server = SetupMockRouterForTest(WithVersion("5.1"))
		Expect(AwgConf.ConfigureOrUpdateInterface(createConf(conf("")), "Wireguard0")).To(MatchError(ContainSubstring("5.2")))
	})

	It("rejects firmware before the ASC3 alpha build", func() {
		server.Close()
		server = SetupMockRouterForTest(WithVersion("5.02.A.10.0-1"))
		Expect(AwgConf.ConfigureOrUpdateInterface(createConf(conf("")), "Wireguard0")).To(MatchError(ContainSubstring("A.11")))
	})

	It("accepts a successful RCI response without a status field", func() {
		server.Close()
		server = SetupMockRouterForTest(WithVersion("5.2"), WithRciBody([]byte(`{"message":"accepted"}`)))
		Expect(checkAWG3Response([]byte(`{"message":"accepted"}`))).To(Succeed())
	})

	It("detects a silently ignored ASC update on readback", func() {
		server.Close()
		server = SetupMockRouterForTest(WithVersion("5.2"), WithRciBody([]byte(`{"message":"accepted"}`)))
		Expect(AwgConf.ConfigureOrUpdateInterface(createConf(conf("")), "Wireguard0")).To(MatchError(ContainSubstring("did not apply")))
	})

	It("rejects a nested router error without updating the peer", func() {
		server.Close()
		server = SetupMockRouterForTest(WithVersion("5.2"), WithRciBody([]byte(`{"status":[{"status":"error","message":"bad ASC"}]}`)))
		Expect(AwgConf.ConfigureOrUpdateInterface(createConf(conf("")), "Wireguard0")).To(MatchError(ContainSubstring("bad ASC")))
		state, err := Interface.GetInterfaceViaRciShowScInterfaces("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Wireguard.Peer).To(BeEmpty())
	})

	It("preserves the legacy AWG 2 command path", func() {
		path := createConf(`[Interface]
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
[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
`)
		commands, err := AwgConf.PlanUpdate(path, "Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(commands).NotTo(BeEmpty())
		Expect(commands[0].Parse).To(Equal("interface Wireguard0 wireguard asc 40 5 95 10 20 1 2 3 4 50 0 0 0 0 0 0"))
	})
})
