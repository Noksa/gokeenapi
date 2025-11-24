package cmd

import (
	"testing"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AddRoutesTestSuite struct {
	CmdTestSuite
}

func TestAddRoutesTestSuite(t *testing.T) {
	suite.Run(t, new(AddRoutesTestSuite))
}

func (s *AddRoutesTestSuite) TestNewAddRoutesCmd() {
	cmd := newAddRoutesCmd()

	assert.Equal(s.T(), CmdAddRoutes, cmd.Use)
	assert.Equal(s.T(), AliasesAddRoutes, cmd.Aliases)
	assert.NotEmpty(s.T(), cmd.Short)
	assert.NotNil(s.T(), cmd.RunE)
}

func (s *AddRoutesTestSuite) TestAddRoutesCmd_Execute() {
	// Set up test config with routes
	config.Cfg.Routes = []config.Route{
		{
			InterfaceID: "Wireguard0",
			BatFileList: config.BatFileList{
				BatFile: []string{},
			},
			BatURLList: config.BatURLList{
				BatURL: []string{},
			},
		},
	}

	cmd := newAddRoutesCmd()
	err := cmd.RunE(cmd, []string{})

	assert.NoError(s.T(), err)
}

func (s *AddRoutesTestSuite) TestAddRoutesCmd_EmptyRoutes() {
	// Empty routes config
	config.Cfg.Routes = []config.Route{}

	cmd := newAddRoutesCmd()
	err := cmd.RunE(cmd, []string{})

	assert.NoError(s.T(), err)
}
