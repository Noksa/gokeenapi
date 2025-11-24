package gokeenrestapi

import (
	"net/http/httptest"

	"github.com/stretchr/testify/suite"
)

// GokeenrestapiTestSuite provides common test setup for gokeenrestapi tests
type GokeenrestapiTestSuite struct {
	suite.Suite
	server *httptest.Server
}

// SetupTest runs before each test to ensure test isolation
func (s *GokeenrestapiTestSuite) SetupTest() {
	s.server = SetupMockRouterForTest()
}

// TearDownTest runs after each test to clean up
func (s *GokeenrestapiTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
}
