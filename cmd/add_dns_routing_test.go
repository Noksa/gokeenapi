package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AddDnsRoutingTestSuite struct {
	CmdTestSuite
}

func TestAddDnsRoutingTestSuite(t *testing.T) {
	suite.Run(t, new(AddDnsRoutingTestSuite))
}

// SetupTest overrides the base SetupTest to use version 5.0.1 for DNS-routing tests
func (s *AddDnsRoutingTestSuite) SetupTest() {
	s.server = gokeenrestapi.SetupMockRouterForTest(gokeenrestapi.WithVersion("5.0.1"))

	err := gokeenrestapi.Common.Auth()
	s.Require().NoError(err)
}

func (s *AddDnsRoutingTestSuite) TestNewAddDnsRoutingCmd() {
	cmd := newAddDnsRoutingCmd()

	assert.Equal(s.T(), CmdAddDnsRouting, cmd.Use)
	assert.Equal(s.T(), AliasesAddDnsRouting, cmd.Aliases)
	assert.NotEmpty(s.T(), cmd.Short)
	assert.NotNil(s.T(), cmd.RunE)
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_Execute() {
	// Create temporary domain files for testing
	tmpDir := s.T().TempDir()

	socialMediaFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(socialMediaFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	streamingFile := filepath.Join(tmpDir, "streaming.txt")
	err = os.WriteFile(streamingFile, []byte("youtube.com\nnetflix.com\n8.8.8.8\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with DNS routing groups
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{socialMediaFile},
					InterfaceID: "Wireguard0",
				},
				{
					Name:        "streaming",
					DomainFile:  []string{streamingFile},
					InterfaceID: "ISP",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.NoError(s.T(), err)
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_EmptyConfiguration() {
	// Empty DNS routing configuration
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err := cmd.RunE(cmd, []string{})

	assert.NoError(s.T(), err)
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_EmptyGroupName() {
	// Create temporary domain file
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("facebook.com\n"), 0644)
	s.Require().NoError(err)

	// Invalid configuration: empty group name
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "",
					DomainFile:  []string{domainFile},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "group name cannot be empty")
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_WhitespaceGroupName() {
	// Create temporary domain file
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("facebook.com\n"), 0644)
	s.Require().NoError(err)

	// Invalid configuration: whitespace-only group name
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "   ",
					DomainFile:  []string{domainFile},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "whitespace")
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_EmptyDomainList() {
	// Invalid configuration: no domain-file or domain-url
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{},
					DomainURL:   []string{},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err := cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "must contain at least one domain-file or domain-url")
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_MalformedDomain() {
	// Create temporary domain file with malformed domain
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("invalid..domain\n"), 0644)
	s.Require().NoError(err)

	// Invalid configuration: malformed domain
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{domainFile},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid domain")
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_InvalidIP() {
	// Create temporary domain file with invalid IP
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("999.999.999.999\n"), 0644)
	s.Require().NoError(err)

	// Invalid configuration: invalid IP address
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "streaming",
					DomainFile:  []string{domainFile},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid domain or IP")
}

func (s *AddDnsRoutingTestSuite) TestAddDnsRoutingCmd_InvalidConfiguration_EmptyInterfaceID() {
	// Create temporary domain file
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("facebook.com\n"), 0644)
	s.Require().NoError(err)

	// Invalid configuration: empty interface ID
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{domainFile},
					InterfaceID: "",
				},
			},
		},
	}

	cmd := newAddDnsRoutingCmd()
	err = cmd.RunE(cmd, []string{})

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "interface ID cannot be empty")
}
