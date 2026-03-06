package cmd

import (
	"os"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common", func() {
	Describe("checkRequiredFields", func() {
		It("should pass with all fields set", func() {
			config.Cfg.Keenetic.URL = "http://192.168.1.1"
			config.Cfg.Keenetic.Login = "admin"
			config.Cfg.Keenetic.Password = "password"

			Expect(checkRequiredFields()).To(Succeed())
		})

		It("should fail when URL is missing", func() {
			config.Cfg.Keenetic.URL = ""
			config.Cfg.Keenetic.Login = "admin"
			config.Cfg.Keenetic.Password = "password"

			err := checkRequiredFields()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("keenetic.url"))
		})

		It("should fail when login is missing", func() {
			config.Cfg.Keenetic.URL = "http://192.168.1.1"
			config.Cfg.Keenetic.Login = ""
			config.Cfg.Keenetic.Password = "password"

			err := checkRequiredFields()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("keenetic.login"))
		})

		It("should fail when password is missing", func() {
			config.Cfg.Keenetic.URL = "http://192.168.1.1"
			config.Cfg.Keenetic.Login = "admin"
			config.Cfg.Keenetic.Password = ""

			err := checkRequiredFields()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("keenetic.password"))
		})
	})

	Describe("RestoreCursor", func() {
		It("should not panic", func() {
			Expect(func() { RestoreCursor() }).NotTo(Panic())
		})
	})

	Describe("confirmAction", func() {
		It("should return true for 'y' input", func() {
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			r, w, _ := os.Pipe()
			os.Stdin = r
			go func() {
				defer func() { _ = w.Close() }()
				_, _ = w.Write([]byte("y\n"))
			}()

			result, err := confirmAction("Test question?")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return false for 'n' input", func() {
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			r, w, _ := os.Pipe()
			os.Stdin = r
			go func() {
				defer func() { _ = w.Close() }()
				_, _ = w.Write([]byte("n\n"))
			}()

			result, err := confirmAction("Test question?")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("should return error on EOF", func() {
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			r, w, _ := os.Pipe()
			os.Stdin = r
			_ = w.Close()

			result, err := confirmAction("Test question?")
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
			Expect(
				strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "canceled"),
			).To(BeTrue())
		})
	})
})
