package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewRootCmd", func() {
	It("should create root command with correct configuration", func() {
		cmd := NewRootCmd()

		Expect(cmd.Use).To(Equal("gokeenapi"))
		Expect(cmd.PersistentPreRunE).NotTo(BeNil())

		configFlag := cmd.PersistentFlags().Lookup("config")
		Expect(configFlag).NotTo(BeNil())
		Expect(configFlag.Value.Type()).To(Equal("string"))
	})

	It("should register all expected subcommands", func() {
		cmd := NewRootCmd()

		subcommands := cmd.Commands()
		Expect(subcommands).NotTo(BeEmpty())

		commandNames := make(map[string]bool)
		for _, subcmd := range subcommands {
			commandNames[subcmd.Use] = true
		}

		expectedCommands := []string{
			CmdShowInterfaces,
			CmdAddRoutes,
			CmdDeleteRoutes,
			CmdAddDnsRecords,
			CmdDeleteDnsRecords,
			CmdAddAwg,
			CmdUpdateAwg,
			CmdDeleteKnownHosts,
			CmdExec,
		}

		for _, expectedCmd := range expectedCommands {
			Expect(commandNames).To(HaveKey(expectedCmd), "Expected command %s not found", expectedCmd)
		}
	})
})
