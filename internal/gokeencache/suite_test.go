package gokeencache

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGokeencache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gokeencache Suite")
}
