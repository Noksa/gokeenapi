package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddRoutes", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes", func() {
		cmd := newAddRoutesCmd()

		Expect(cmd.Use).To(Equal(CmdAddRoutes))
		Expect(cmd.Aliases).To(Equal(AliasesAddRoutes))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	It("should execute with configured routes", func() {
		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "Wireguard0",
				BatFileList: config.BatFileList{BatFile: []string{}},
				BatURLList:  config.BatURLList{BatURL: []string{}},
			},
		}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should handle empty routes config", func() {
		config.Cfg.Routes = []config.Route{}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})
})
