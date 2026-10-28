package exporter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestExporter(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "Exporter Suite") }
