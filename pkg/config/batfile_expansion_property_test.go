package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

var _ = Describe("Property: Bat-File Expansion Idempotency", func() {
	It("should produce identical results on repeated expansion", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			cfg := genGokeenapiConfig().Draw(t, "config")

			for i := range cfg.Routes {
				for j, batFile := range cfg.Routes[i].BatFile {
					if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
						batListContent := BatFileList{
							BatFile: []string{
								"/expanded/file1.bat",
								"/expanded/file2.bat",
							},
						}

						yamlBytes, err := yaml.Marshal(&batListContent)
						Expect(err).NotTo(HaveOccurred())

						yamlPath := filepath.Join(tmpDir, filepath.Base(batFile))
						err = os.WriteFile(yamlPath, yamlBytes, 0644)
						Expect(err).NotTo(HaveOccurred())

						cfg.Routes[i].BatFile[j] = filepath.Base(batFile)
					}
				}
			}

			configPath := filepath.Join(tmpDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			firstExpansion := make([][]string, len(Cfg.Routes))
			for i := range Cfg.Routes {
				firstExpansion[i] = make([]string, len(Cfg.Routes[i].BatFile))
				copy(firstExpansion[i], Cfg.Routes[i].BatFile)
			}

			err = expandBatLists(configPath)
			Expect(err).NotTo(HaveOccurred())

			for i := range Cfg.Routes {
				Expect(Cfg.Routes[i].BatFile).To(HaveLen(len(firstExpansion[i])),
					"expansion not idempotent: route %d has different lengths", i)

				for j := range firstExpansion[i] {
					Expect(Cfg.Routes[i].BatFile[j]).To(Equal(firstExpansion[i][j]),
						"expansion not idempotent: route %d, file %d", i, j)
				}
			}
		})
	})
})

var _ = Describe("Property: Bat-URL Expansion Idempotency", func() {
	It("should produce identical results on repeated expansion", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			cfg := genGokeenapiConfig().Draw(t, "config")

			yamlFilesCreated := make(map[string]bool)

			for i := range cfg.Routes {
				for j, batFile := range cfg.Routes[i].BatFile {
					if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
						baseName := filepath.Base(batFile)
						if !yamlFilesCreated[baseName] {
							batListContent := BatFileList{
								BatFile: []string{
									"/expanded/file1.bat",
									"/expanded/file2.bat",
								},
							}

							yamlBytes, err := yaml.Marshal(&batListContent)
							Expect(err).NotTo(HaveOccurred())

							yamlPath := filepath.Join(tmpDir, baseName)
							err = os.WriteFile(yamlPath, yamlBytes, 0644)
							Expect(err).NotTo(HaveOccurred())
							yamlFilesCreated[baseName] = true
						}
						cfg.Routes[i].BatFile[j] = filepath.Base(batFile)
					}
				}

				for j, batURL := range cfg.Routes[i].BatURL {
					if filepath.Ext(batURL) == ".yaml" || filepath.Ext(batURL) == ".yml" {
						baseName := filepath.Base(batURL)
						if !yamlFilesCreated[baseName] {
							batURLListContent := BatURLList{
								BatURL: []string{
									"https://example.com/expanded/url1.bat",
									"https://example.com/expanded/url2.bat",
								},
							}

							yamlBytes, err := yaml.Marshal(&batURLListContent)
							Expect(err).NotTo(HaveOccurred())

							yamlPath := filepath.Join(tmpDir, baseName)
							err = os.WriteFile(yamlPath, yamlBytes, 0644)
							Expect(err).NotTo(HaveOccurred())
							yamlFilesCreated[baseName] = true
						}
						cfg.Routes[i].BatURL[j] = filepath.Base(batURL)
					}
				}
			}

			configPath := filepath.Join(tmpDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			firstExpansion := make([][]string, len(Cfg.Routes))
			for i := range Cfg.Routes {
				firstExpansion[i] = make([]string, len(Cfg.Routes[i].BatURL))
				copy(firstExpansion[i], Cfg.Routes[i].BatURL)
			}

			err = expandBatLists(configPath)
			Expect(err).NotTo(HaveOccurred())

			for i := range Cfg.Routes {
				Expect(Cfg.Routes[i].BatURL).To(HaveLen(len(firstExpansion[i])),
					"expansion not idempotent: route %d has different lengths", i)

				for j := range firstExpansion[i] {
					Expect(Cfg.Routes[i].BatURL[j]).To(Equal(firstExpansion[i][j]),
						"expansion not idempotent: route %d, url %d", i, j)
				}
			}
		})
	})
})

var _ = Describe("Property: Bat-File and Bat-URL Expansion Independence", func() {
	It("should expand bat-file and bat-url independently", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			batFileList := []string{
				"/path/to/file1.bat",
				"/path/to/file2.bat",
			}
			batURLList := []string{
				"https://example.com/url1.bat",
				"https://example.com/url2.bat",
			}

			combinedContent := BatLists{
				BatFileList: BatFileList{BatFile: batFileList},
				BatURLList:  BatURLList{BatURL: batURLList},
			}

			yamlBytes, err := yaml.Marshal(&combinedContent)
			Expect(err).NotTo(HaveOccurred())

			combinedPath := filepath.Join(tmpDir, "combined.yaml")
			err = os.WriteFile(combinedPath, yamlBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			// Test 1: Reference in bat-file only
			cfg1 := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{"combined.yaml"}},
					},
				},
			}

			configPath1 := filepath.Join(tmpDir, "config1.yaml")
			configBytes1, err := yaml.Marshal(&cfg1)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath1, configBytes1, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath1)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(len(batFileList)),
				"bat-file not expanded correctly")
			Expect(Cfg.Routes[0].BatURL).To(BeEmpty(),
				"bat-url should not be expanded when only in bat-file")

			// Test 2: Reference in bat-url only
			cfg2 := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatURLList:  BatURLList{BatURL: []string{"combined.yaml"}},
					},
				},
			}

			configPath2 := filepath.Join(tmpDir, "config2.yaml")
			configBytes2, err := yaml.Marshal(&cfg2)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath2, configBytes2, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath2)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(BeEmpty(),
				"bat-file should not be expanded when only in bat-url")
			Expect(Cfg.Routes[0].BatURL).To(HaveLen(len(batURLList)),
				"bat-url not expanded correctly")
		})
	})
})

var _ = Describe("Property: YAML File Caching", func() {
	It("should correctly expand same YAML file referenced in both arrays", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			batFileList := []string{"/path/to/file1.bat", "/path/to/file2.bat"}
			batURLList := []string{"https://example.com/url1.bat", "https://example.com/url2.bat"}

			combinedContent := BatLists{
				BatFileList: BatFileList{BatFile: batFileList},
				BatURLList:  BatURLList{BatURL: batURLList},
			}

			yamlBytes, err := yaml.Marshal(&combinedContent)
			Expect(err).NotTo(HaveOccurred())

			combinedPath := filepath.Join(tmpDir, "combined.yaml")
			err = os.WriteFile(combinedPath, yamlBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{URL: "http://192.168.1.1", Login: "admin", Password: "password"},
				Routes: []Route{
					{
						InterfaceID: "Wireguard0",
						BatFileList: BatFileList{BatFile: []string{"combined.yaml"}},
						BatURLList:  BatURLList{BatURL: []string{"combined.yaml"}},
					},
				},
			}

			configPath := filepath.Join(tmpDir, "config.yaml")
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Routes[0].BatFile).To(HaveLen(len(batFileList)))
			Expect(Cfg.Routes[0].BatURL).To(HaveLen(len(batURLList)))

			for i, expected := range batFileList {
				Expect(Cfg.Routes[0].BatFile[i]).To(Equal(expected))
			}
			for i, expected := range batURLList {
				Expect(Cfg.Routes[0].BatURL[i]).To(Equal(expected))
			}
		})
	})
})
