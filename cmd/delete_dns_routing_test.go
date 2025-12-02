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

	// Verify the social-media group was deleted
	groups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
	assert.NoError(s.T(), err)
	_, exists := groups["social-media"]
	assert.False(s.T(), exists, "social-media group should be deleted")
	// streaming group should still exist (not in config)
	_, exists = groups["streaming"]
	assert.True(s.T(), exists, "streaming group should still exist")
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

	// Verify all groups were deleted
	groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), groupsAfter, "All DNS routing groups should be deleted")
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_NoGroupsOnRouter() {
	// Create a router with no DNS routing groups
	s.server = gokeenrestapi.SetupMockRouterForTest(
		gokeenrestapi.WithVersion("5.0.1"),
	)
	err := gokeenrestapi.Common.Auth()
	s.Require().NoError(err)

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_DeletesAllGroups() {
	// Create temporary domain files
	tmpDir := s.T().TempDir()

	socialFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(socialFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	streamingFile := filepath.Join(tmpDir, "streaming.txt")
	err = os.WriteFile(streamingFile, []byte("youtube.com\nnetflix.com\n8.8.8.8\n"), 0644)
	s.Require().NoError(err)

	// Set up test config with both groups
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

	// Verify router has groups before deletion
	groupsBefore, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), groupsBefore, "Router should have groups before deletion")

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)

	// Verify all groups were deleted
	groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), groupsAfter, "All groups should be deleted")
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

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_WithInterfaceId() {
	// Create temporary domain files
	tmpDir := s.T().TempDir()

	socialFile := filepath.Join(tmpDir, "social-media.txt")
	err := os.WriteFile(socialFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)
	s.Require().NoError(err)

	// Set up test config - but we'll use --interface-id flag instead
	config.Cfg.DNS = config.DNS{
		Routes: config.DnsRoutes{
			Groups: []config.DnsRoutingGroup{},
		},
	}

	cmd := newDeleteDnsRoutingCmd()
	_ = cmd.Flags().Set("force", "true")
	_ = cmd.Flags().Set("interface-id", "Wireguard0")

	err = cmd.RunE(cmd, []string{})
	assert.NoError(s.T(), err)

	// Verify only Wireguard0 group was deleted
	groups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
	assert.NoError(s.T(), err)
	_, exists := groups["social-media"]
	assert.False(s.T(), exists, "social-media group (Wireguard0) should be deleted")
	// streaming group on ISP should still exist
	_, exists = groups["streaming"]
	assert.True(s.T(), exists, "streaming group (ISP) should still exist")
}

func (s *DeleteDnsRoutingTestSuite) TestDeleteDnsRoutingCmd_InterfaceIdFlag() {
	cmd := newDeleteDnsRoutingCmd()

	interfaceIdFlag := cmd.Flags().Lookup("interface-id")
	assert.NotNil(s.T(), interfaceIdFlag)
	assert.Equal(s.T(), "string", interfaceIdFlag.Value.Type())
	assert.Equal(s.T(), "", interfaceIdFlag.DefValue)
}
