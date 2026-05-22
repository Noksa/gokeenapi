package cmd

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// When executeCommandsForConfig spawns os.Executable() as a subprocess during
	// tests, that subprocess is this same test binary. Exit immediately with failure
	// to simulate a failing command and break the recursion.
	if os.Getenv("GOKEENAPI_TEST_SUBPROCESS") != "" {
		os.Exit(1)
	}
	// Mark child processes so they exit fast instead of re-running the test suite.
	if err := os.Setenv("GOKEENAPI_TEST_SUBPROCESS", "1"); err != nil {
		panic("failed to set GOKEENAPI_TEST_SUBPROCESS: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}
