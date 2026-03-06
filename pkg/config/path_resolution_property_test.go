package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

var _ = Describe("Property: Bat-File Path Resolution Consistency", func() {
	It("should resolve paths consistently regardless of working directory", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			relativePath := genRelativePath().Draw(t, "relativePath")

			configDir := filepath.Join(tmpDir, "configs")
			err := os.MkdirAll(configDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			batListPath := filepath.Join(configDir, relativePath)
			err = os.MkdirAll(filepath.Dir(batListPath), 0755)
			Expect(err).NotTo(HaveOccurred())

			batListContent := BatFileList{
				BatFile: []string{"/test/file1.bat", "/test/file2.bat"},
			}
			yamlBytes, err := yaml.Marshal(&batListContent)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(batListPath, yamlBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{relativePath}},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			originalWd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			firstExpansion := make([]string, len(Cfg.Routes[0].BatFile))
			copy(firstExpansion, Cfg.Routes[0].BatFile)

			err = os.Chdir(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(originalWd) }()

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			secondExpansion := Cfg.Routes[0].BatFile

			Expect(secondExpansion).To(HaveLen(len(firstExpansion)),
				"path resolution not consistent: different lengths")

			for i := range firstExpansion {
				Expect(secondExpansion[i]).To(Equal(firstExpansion[i]),
					"path resolution not consistent: file %d", i)
			}
		})
	})
})

var _ = Describe("Property: Regular Bat-Files Resolved Relative to Config", func() {
	It("should resolve relative bat-file paths relative to config directory", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			relativeBatPath := rapid.StringMatching(`[a-z]{3,10}/[a-z]{3,10}\.bat`).Draw(t, "relativeBatPath")

			configDir := filepath.Join(tmpDir, "configs", rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "configSubdir"))
			err := os.MkdirAll(configDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			batFilePath := filepath.Join(configDir, relativeBatPath)
			err = os.MkdirAll(filepath.Dir(batFilePath), 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(batFilePath, []byte("route add 1.1.1.1 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{relativeBatPath}},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(1))

			resolvedPath := Cfg.Routes[0].BatFile[0]
			expectedPath := filepath.Join(configDir, relativeBatPath)

			Expect(resolvedPath).To(Equal(expectedPath))
			Expect(filepath.IsAbs(resolvedPath)).To(BeTrue(), "resolved path is not absolute: %s", resolvedPath)

			_, err = os.Stat(resolvedPath)
			Expect(err).NotTo(HaveOccurred(), "resolved path does not exist: %s", resolvedPath)
		})
	})
})

var _ = Describe("Property: Regular Domain-Files Resolved Relative to Config", func() {
	It("should resolve relative domain-file paths relative to config directory", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			relativeDomainPath := rapid.StringMatching(`[a-z]{3,10}/[a-z]{3,10}\.txt`).Draw(t, "relativeDomainPath")

			configDir := filepath.Join(tmpDir, "configs", rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "configSubdir"))
			err := os.MkdirAll(configDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			domainFilePath := filepath.Join(configDir, relativeDomainPath)
			err = os.MkdirAll(filepath.Dir(domainFilePath), 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(domainFilePath, []byte("example.com\ntest.com"), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				DNS: DNS{
					Routes: DnsRoutes{
						Groups: []DnsRoutingGroup{
							{
								Name:        "test-group",
								DomainFile:  []string{relativeDomainPath},
								InterfaceID: "Wireguard0",
							},
						},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.DNS.Routes.Groups[0].DomainFile).To(HaveLen(1))

			resolvedPath := Cfg.DNS.Routes.Groups[0].DomainFile[0]
			expectedPath := filepath.Join(configDir, relativeDomainPath)

			Expect(resolvedPath).To(Equal(expectedPath))
			Expect(filepath.IsAbs(resolvedPath)).To(BeTrue(), "resolved path is not absolute: %s", resolvedPath)

			_, err = os.Stat(resolvedPath)
			Expect(err).NotTo(HaveOccurred(), "resolved path does not exist: %s", resolvedPath)
		})
	})
})

var _ = Describe("Property: Absolute Paths Preserved Unchanged", func() {
	It("should not modify absolute bat-file paths", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			absoluteBatPath := filepath.Join(tmpDir, rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "dir"), rapid.StringMatching(`[a-z]{3,10}\.bat`).Draw(t, "file"))

			err := os.MkdirAll(filepath.Dir(absoluteBatPath), 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(absoluteBatPath, []byte("route add 1.1.1.1 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())

			configDir := filepath.Join(tmpDir, "configs")
			err = os.MkdirAll(configDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{absoluteBatPath}},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(1))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal(absoluteBatPath))
		})
	})
})

var _ = Describe("Property: Paths in YAML Lists Resolved Relative to YAML File", func() {
	It("should resolve paths relative to YAML file directory, not config directory", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			relativeBatPath1 := rapid.StringMatching(`[a-z]{3,10}\.bat`).Draw(t, "bat1")
			relativeBatPath2 := rapid.StringMatching(`[a-z]{3,10}\.bat`).Draw(t, "bat2")

			configDir := filepath.Join(tmpDir, "configs")
			yamlDir := filepath.Join(configDir, "batfiles")
			err := os.MkdirAll(yamlDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			batFile1 := filepath.Join(yamlDir, relativeBatPath1)
			batFile2 := filepath.Join(yamlDir, relativeBatPath2)
			err = os.WriteFile(batFile1, []byte("route add 1.1.1.1 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(batFile2, []byte("route add 2.2.2.2 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())

			yamlListPath := filepath.Join(yamlDir, "common.yaml")
			batList := BatFileList{
				BatFile: []string{relativeBatPath1, relativeBatPath2},
			}
			yamlBytes, err := yaml.Marshal(&batList)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(yamlListPath, yamlBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{"batfiles/common.yaml"}},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal(batFile1))
			Expect(Cfg.Routes[0].BatFile[1]).To(Equal(batFile2))

			for i, path := range Cfg.Routes[0].BatFile {
				_, err := os.Stat(path)
				Expect(err).NotTo(HaveOccurred(), "resolved path %d does not exist: %s", i, path)
			}
		})
	})
})

var _ = Describe("Property: Mixed Absolute and Relative Paths", func() {
	It("should handle both absolute and relative paths correctly", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			configDir := filepath.Join(tmpDir, "configs")
			err := os.MkdirAll(configDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			absoluteDir := filepath.Join(tmpDir, "absolute")
			err = os.MkdirAll(absoluteDir, 0755)
			Expect(err).NotTo(HaveOccurred())
			absoluteBatPath := filepath.Join(absoluteDir, "absolute.bat")
			err = os.WriteFile(absoluteBatPath, []byte("route add 1.1.1.1 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())

			relativeDir := filepath.Join(configDir, "relative")
			err = os.MkdirAll(relativeDir, 0755)
			Expect(err).NotTo(HaveOccurred())
			relativeBatPath := "relative/relative.bat"
			err = os.WriteFile(filepath.Join(configDir, relativeBatPath), []byte("route add 2.2.2.2 mask 255.255.255.255 0.0.0.0"), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{absoluteBatPath, relativeBatPath}},
					},
				},
			}

			configPath := filepath.Join(configDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal(absoluteBatPath),
				"absolute path was modified")

			expectedRelativePath := filepath.Join(configDir, relativeBatPath)
			Expect(Cfg.Routes[0].BatFile[1]).To(Equal(expectedRelativePath),
				"relative path not resolved correctly")

			for i, path := range Cfg.Routes[0].BatFile {
				_, err := os.Stat(path)
				Expect(err).NotTo(HaveOccurred(), "resolved path %d does not exist: %s", i, path)
			}
		})
	})
})
