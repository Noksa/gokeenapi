package gokeenrestapi

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGokeenrestapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gokeenrestapi Suite")
}
