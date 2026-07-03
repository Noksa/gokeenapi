package gokeenrestapi

import (
	"io"
	"os"

	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// captureStdout captures stdout output from fn execution
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

var _ = Describe("warnIfAWG2Unsupported", func() {
	BeforeEach(func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: ""}
		})
	})

	It("should warn when firmware < 5.1 and AWG 2.0 params present", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.0.1"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
				S3: "50", S4: "60",
			})
		})

		Expect(output).To(ContainSubstring("WARNING"))
		Expect(output).To(ContainSubstring("AWG 2.0"))
		Expect(output).To(ContainSubstring("5.1"))
		Expect(output).To(ContainSubstring("5.0.1"))
	})

	It("should warn on firmware 4.x with AWG 2.0 params", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "4.3.6.3"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
				I1: "42",
			})
		})

		Expect(output).To(ContainSubstring("WARNING"))
		Expect(output).To(ContainSubstring("4.3.6.3"))
	})

	It("should not warn when firmware >= 5.1", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.1.0"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
				S3: "50", S4: "60", I1: "1", I2: "2", I3: "3", I4: "4", I5: "5",
			})
		})

		Expect(output).To(BeEmpty())
	})

	It("should not warn when firmware is 5.1 beta", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.1 Beta 4"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
				S3: "77",
			})
		})

		Expect(output).To(BeEmpty())
	})

	It("should not warn when no AWG 2.0 params are present", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "4.3.6.3"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
			})
		})

		Expect(output).To(BeEmpty())
	})

	It("should not warn when version is empty", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: ""}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{S3: "50"})
		})

		Expect(output).To(BeEmpty())
	})

	It("should not warn on firmware 5.2+", func() {
		gokeencache.UpdateRuntimeConfig(func(runtime *config.Runtime) {
			runtime.RouterInfo.Version = gokeenrestapimodels.Version{Title: "5.2.3"}
		})

		output := captureStdout(func() {
			warnIfAWG2Unsupported(ascParams{
				Jc: "5", Jmin: "10", Jmax: "100", S1: "1", S2: "2",
				H1: "1", H2: "2", H3: "3", H4: "4",
				S3: "1", S4: "2", I1: "3", I2: "4", I3: "5", I4: "6", I5: "7",
			})
		})

		Expect(output).To(BeEmpty())
	})
})
