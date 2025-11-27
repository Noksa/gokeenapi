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

type DeleteDnsRoutingTestSuite struct {
	CmdTestSuite
}

func TestDeleteDnsRoutingTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteDnsRoutingTestSuite))
}

// SetupTest overrides the base SetupTest to use version 5.0.1 for DNS-routing tests
func (s *DeleteDnsRoutingTestSuite) SetupTest() {
	// Create mock router with DNS-routing groups already configured
	s.server = gokeenrestapi.SetupMockRouterForTest(
		gokeenrestapi.WithVersion("5.0.1"),
		gokeenrestapi.WithDnsRoutingGroups(
			[]gokeenrestapi.MockDnsRoutingGroup{
				{
					Name:    "social-media",
					Domains: []string{"facebook.com", "instagram.com", "twitter.com"},
				},
				{
					Name:    "streaming",
					Domains: []string{"youtube.com", "netflix.com", "8.8.8.8"},
				},
			},
			[]gokeenrestapi.MockDnsProxyRoute{
				{
					GroupName:   "social-media",
					InterfaceID: "Wireguard0",
					Mode:        "auto",
				},
				{
					GroupName:   "streaming",
					InterfaceID: "ISP",
					Mode:        "auto",
				},
			},
		),
	)

	err := gokeenrestapi.Common.Auth()
	s.Require().NoError(err)
}

func (s *DeleteDnsRoutingTestSuite) TestNewDeleteDnsRoutingCmd() {
	cmd := newDeleteDnsRoutingCmd()

	assert.Equal(s.T(), CmdDeleteDnsRouting, cmd.Use)
	assert.Equal(s.T(), AliasesDeleteDnsRouting, cmd.Aliases)
	assert.NotEmpty(s.T(), cmd.Short)
	assert.NotNil(s.T(), cmd.RunE)

	// Verify --force flag exists
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(s.T(), forceFlag)
	assert.Equal(s.T(), "bool", forceFlag.Value.Type())
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_WithForce() {
	// Create temporary domain file
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(domainFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with DNS routing groups that match the mock router
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

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_WithForce_MultipleGroups() {
	// Create temporary domain files
	tmpDir := s.T().TempDir()

	socialFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(socialFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	streamingFile := filepath.Join(tmpDir, "streaming.txt")
	err = os.WriteFile(streamingFile, []byte("youtube.com\nnetflix.com\n8.8.8.8\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with multiple DNS routing groups
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{socialFile},
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

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_EmptyConfiguration() {
	// Empty DNS routing configuration
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{},
		},
	}

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err := cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_NoMatchingGroups() {
	// Create temporary domain file
	tmpDir := s.T().TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err := os.WriteFile(domainFile, []byte("example.com\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with groups that don't exist in the router
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "non-existent-group",
					DomainFile:  []string{domainFile},
					InterfaceID: "Wireguard0",
				},
			},
		},
	}

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_ForceFlagDefaultValue() {
	cmd := newDeleteDnsRoutingCmd()

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(s.T(), forceFlag)
	assert.Equal(s.T(), "false", forceFlag.DefValue)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_PartialMatch() {
	// Create temporary domain files
	tmpDir := s.T().TempDir()

	socialFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(socialFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	nonExistentFile := filepath.Join(tmpDir, "non-existent.txt")
	err = os.WriteFile(nonExistentFile, []byte("example.com\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with one matching and one non-matching group
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{
				{
					Name:        "social-media",
					DomainFile:  []string{socialFile},
					InterfaceID: "Wireguard0",
				},
				{
					Name:        "non-existent-group",
					DomainFile:  []string{nonExistentFile},
					InterfaceID: "ISP",
				},
			},
		},
	}

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}
