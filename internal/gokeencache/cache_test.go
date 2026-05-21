package gokeencache

import (
	"os"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetGokeenDir", func() {
	It("should create directory with restricted permissions", func() {
		tmpDir, err := os.MkdirTemp("", "gokeencache-perm-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = os.RemoveAll(tmpDir)
			config.Cfg.DataDir = ""
		})
		config.Cfg.DataDir = tmpDir

		gokeenDir, err := GetGokeenDir()
		Expect(err).NotTo(HaveOccurred())

		info, err := os.Stat(gokeenDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0700)))
	})
})

var _ = Describe("DomainValidationCache", func() {
	It("should return not cached for unknown domain", func() {
		_, cached := GetDomainValidation("example.com")
		Expect(cached).To(BeFalse())
	})

	It("should cache valid domain", func() {
		domain := "cached-valid.com"
		SetDomainValidation(domain, true)

		valid, cached := GetDomainValidation(domain)
		Expect(cached).To(BeTrue())
		Expect(valid).To(BeTrue())
	})

	It("should cache invalid domain", func() {
		domain := "invalid..domain"
		SetDomainValidation(domain, false)

		valid, cached := GetDomainValidation(domain)
		Expect(cached).To(BeTrue())
		Expect(valid).To(BeFalse())
	})
})

var _ = Describe("URLContentWithChecksum", func() {
	var (
		url     string
		content string
		ttl     time.Duration
	)

	BeforeEach(func() {
		tmpDir, err := os.MkdirTemp("", "gokeencache-test-*")
		Expect(err).NotTo(HaveOccurred())
		config.Cfg.DataDir = tmpDir
		DeferCleanup(func() {
			_ = os.RemoveAll(tmpDir)
			config.Cfg.DataDir = ""
		})

		url = "https://example.com/domains.txt"
		content = "example.com\ntest.com\n"
		ttl = 5 * time.Minute
	})

	It("should store and retrieve content", func() {
		Expect(SetURLContent(url, content, ttl)).To(Succeed())

		retrieved, ok := GetURLContent(url)
		Expect(ok).To(BeTrue())
		Expect(retrieved).To(Equal(content))
	})

	It("should compute checksum", func() {
		Expect(SetURLContent(url, content, ttl)).To(Succeed())

		checksum := GetURLChecksum(url)
		Expect(checksum).NotTo(BeEmpty())
	})

	It("should change checksum when content changes", func() {
		Expect(SetURLContent(url, content, ttl)).To(Succeed())
		checksum1 := GetURLChecksum(url)

		newContent := "example.com\ntest.com\nnew.com\n"
		Expect(SetURLContent(url, newContent, ttl)).To(Succeed())
		checksum2 := GetURLChecksum(url)

		Expect(checksum2).NotTo(Equal(checksum1))
	})

	It("should return error when directory is not writable", func() {
		// Make the data dir read-only so writes fail
		Expect(os.Chmod(config.Cfg.DataDir, 0500)).To(Succeed())
		DeferCleanup(func() {
			_ = os.Chmod(config.Cfg.DataDir, 0700)
		})

		// GetGokeenDir will try to MkdirAll .gokeenapi inside a read-only dir
		err := SetURLContent(url, content, ttl)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ComputeChecksum", func() {
	It("should produce same checksum for same content", func() {
		content := []byte("example.com\ntest.com\n")
		Expect(ComputeChecksum(content)).To(Equal(ComputeChecksum(content)))
	})

	It("should produce different checksum for different content", func() {
		content1 := []byte("example.com\ntest.com\n")
		content2 := []byte("different.com\n")
		Expect(ComputeChecksum(content1)).NotTo(Equal(ComputeChecksum(content2)))
	})
})
