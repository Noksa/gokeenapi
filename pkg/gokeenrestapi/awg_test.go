package gokeenrestapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AwgTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (s *AwgTestSuite) SetupTest() {
	// Use unified mock with custom SC interface configuration
	// Set Jc to 40 (different from config file's 50) to trigger update
	s.server = SetupMockRouterForTest(
		WithScInterfaces(map[string]MockScInterface{
			"Wireguard0": {
				Description: "Test WireGuard interface",
				IP: MockIP{
					Address: "10.0.0.1/24",
				},
				Wireguard: MockWireguard{
					Asc: MockAsc{
						Jc:   "40", // Different from config to trigger update
						Jmin: "5",
						Jmax: "95",
						S1:   "10",
						S2:   "20",
						H1:   "1",
						H2:   "2",
						H3:   "3",
						H4:   "4",
					},
					Peer: []MockPeer{},
				},
			},
		}),
	)
}

func (s *AwgTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *AwgTestSuite) createTestWireGuardConfig() string {
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

	tmpDir := s.T().TempDir()
	confPath := filepath.Join(tmpDir, "test.conf")

	err := os.WriteFile(confPath, []byte(confContent), 0644)
	s.Require().NoError(err, "Failed to create test config file")

	return confPath
}

func TestAwgTestSuite(t *testing.T) {
	suite.Run(t, new(AwgTestSuite))
}

func (s *AwgTestSuite) TestConfigureOrUpdateInterface() {
	confPath := s.createTestWireGuardConfig()

	// Test with existing interface - this should work now
	err := AwgConf.ConfigureOrUpdateInterface(confPath, "Wireguard0")
	s.NoError(err)
}

func (s *AwgTestSuite) TestConfigureOrUpdateInterfaceNonExistent() {
	confPath := s.createTestWireGuardConfig()

	// Test with non-existent interface
	err := AwgConf.ConfigureOrUpdateInterface(confPath, "NonExistentInterface")
	s.Error(err)

	// The error should be about interface not found
	s.Contains(err.Error(), "doesn't have interface")
}

func (s *AwgTestSuite) TestConfigureOrUpdateInterfaceEmptyPath() {
	err := AwgConf.ConfigureOrUpdateInterface("", "Wireguard0")
	s.Error(err)
	s.Equal("conf-file flag is required", err.Error())
}
