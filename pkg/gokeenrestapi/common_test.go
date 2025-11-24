package gokeenrestapi

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CheckRouterModeTestSuite struct {
	suite.Suite
}

func TestCheckRouterModeTestSuite(t *testing.T) {
	suite.Run(t, new(CheckRouterModeTestSuite))
}

func (s *CheckRouterModeTestSuite) TestCheckRouterMode_RouterMode() {
	server := SetupMockRouterForTest(
		WithSystemMode(MockSystemMode{
			Active:   "router",
			Selected: "router",
		}),
	)
	defer server.Close()

	active, selected, err := Common.CheckRouterMode()
	s.NoError(err)
	s.Equal("router", active)
	s.Equal("router", selected)
}

func (s *CheckRouterModeTestSuite) TestCheckRouterMode_ExtenderMode() {
	server := SetupMockRouterForTest(
		WithSystemMode(MockSystemMode{
			Active:   "extender",
			Selected: "extender",
		}),
	)
	defer server.Close()

	active, selected, err := Common.CheckRouterMode()
	s.Error(err)
	s.Equal("extender", active)
	s.Equal("extender", selected)
	s.Contains(err.Error(), "router is not in router mode")
}

func (s *CheckRouterModeTestSuite) TestCheckRouterMode_MixedMode() {
	server := SetupMockRouterForTest(
		WithSystemMode(MockSystemMode{
			Active:   "router",
			Selected: "extender",
		}),
	)
	defer server.Close()

	active, selected, err := Common.CheckRouterMode()
	s.Error(err)
	s.Equal("router", active)
	s.Equal("extender", selected)
	s.Contains(err.Error(), "router is not in router mode")
}
