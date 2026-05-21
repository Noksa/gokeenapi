package gokeenrestapi

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

var _ = Describe("EnsureSaveConfigAtEnd", func() {
	saveCmd := "system configuration save"

	makeReqs := func(cmds ...string) []gokeenrestapimodels.ParseRequest {
		reqs := make([]gokeenrestapimodels.ParseRequest, len(cmds))
		for i, c := range cmds {
			reqs[i] = gokeenrestapimodels.ParseRequest{Parse: c}
		}
		return reqs
	}

	It("adds save command to an empty slice", func() {
		result := Common.EnsureSaveConfigAtEnd(nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Parse).To(Equal(saveCmd))
	})

	It("adds save command to the end when absent", func() {
		input := makeReqs("ip route add", "interface up")
		result := Common.EnsureSaveConfigAtEnd(input)
		Expect(result).To(HaveLen(3))
		Expect(result[2].Parse).To(Equal(saveCmd))
		Expect(result[0].Parse).To(Equal("ip route add"))
		Expect(result[1].Parse).To(Equal("interface up"))
	})

	It("moves save command to end when it is in the middle", func() {
		input := makeReqs("ip route add", saveCmd, "interface up")
		result := Common.EnsureSaveConfigAtEnd(input)
		Expect(result).To(HaveLen(3))
		Expect(result[2].Parse).To(Equal(saveCmd))
		Expect(result[0].Parse).To(Equal("ip route add"))
		Expect(result[1].Parse).To(Equal("interface up"))
	})

	It("keeps save command at end without duplication when already last", func() {
		input := makeReqs("ip route add", saveCmd)
		result := Common.EnsureSaveConfigAtEnd(input)
		Expect(result).To(HaveLen(2))
		Expect(result[1].Parse).To(Equal(saveCmd))
		saveCount := 0
		for _, r := range result {
			if r.Parse == saveCmd {
				saveCount++
			}
		}
		Expect(saveCount).To(Equal(1))
	})

	It("deduplicates when multiple save commands are present", func() {
		input := makeReqs(saveCmd, "ip route add", saveCmd)
		result := Common.EnsureSaveConfigAtEnd(input)
		Expect(result[len(result)-1].Parse).To(Equal(saveCmd))
		saveCount := 0
		for _, r := range result {
			if r.Parse == saveCmd {
				saveCount++
			}
		}
		Expect(saveCount).To(Equal(1))
	})
})

var _ = Describe("SaveConfig", func() {
	AfterEach(func() {
		CleanupTestConfig()
		restyClient = nil
		restyClientOnce = sync.Once{}
	})

	It("executes the save config command against the router", func() {
		server := SetupMockRouterForTest()
		DeferCleanup(server.Close)

		err := Common.SaveConfig()
		Expect(err).NotTo(HaveOccurred())
	})
})
