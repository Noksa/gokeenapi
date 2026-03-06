package cmd

import (
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SchedulerBatFileExpansion", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
	})

	It("should expand bat-file lists from YAML references", func() {
		batFileListPath := writeTempFile(tmpDir, "common-routes.yaml", `bat-file:
  - /path/to/route1.bat
  - /path/to/route2.bat
  - /path/to/route3.bat
`)

		routerConfigPath := filepath.Join(tmpDir, "router.yaml")
		Expect(os.WriteFile(routerConfigPath, []byte(`keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - common-routes.yaml
      - /path/to/extra-route.bat
`), 0644)).To(Succeed())

		schedulerConfigPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Test task with expanded bat-files"
    commands:
      - add-routes
    configs:
      - `+routerConfigPath+`
    interval: "1h"
`)

		schedulerCfg, err := config.LoadSchedulerConfig(schedulerConfigPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(schedulerCfg.Tasks).To(HaveLen(1))

		task := schedulerCfg.Tasks[0]
		Expect(task.Name).To(Equal("Test task with expanded bat-files"))
		Expect(task.Commands).To(Equal([]string{"add-routes"}))
		Expect(task.Configs).To(Equal([]string{routerConfigPath}))
		Expect(task.Interval).To(Equal("1h"))

		Expect(config.LoadConfig(routerConfigPath)).To(Succeed())
		Expect(config.Cfg.Routes).To(HaveLen(1))

		route := config.Cfg.Routes[0]
		Expect(route.InterfaceID).To(Equal("Wireguard0"))
		Expect(route.BatFile).To(Equal([]string{
			"/path/to/route1.bat",
			"/path/to/route2.bat",
			"/path/to/route3.bat",
			"/path/to/extra-route.bat",
		}))

		_ = batFileListPath // used via YAML reference
	})

	It("should validate scheduler config with bat-file expansion", func() {
		writeTempFile(tmpDir, "routes.yaml", `bat-file:
  - /path/to/route1.bat
`)

		routerConfigPath := filepath.Join(tmpDir, "router.yaml")
		Expect(os.WriteFile(routerConfigPath, []byte(`keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - routes.yaml
`), 0644)).To(Succeed())

		schedulerConfigPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Test validation"
    commands:
      - add-routes
    configs:
      - `+routerConfigPath+`
    interval: "3h"
`)

		schedulerCfg, err := config.LoadSchedulerConfig(schedulerConfigPath)
		Expect(err).NotTo(HaveOccurred())

		Expect(validateTask(schedulerCfg.Tasks[0])).To(Succeed())

		_, err = os.Stat(routerConfigPath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should expand nested YAML references", func() {
		writeTempFile(tmpDir, "list1.yaml", `bat-file:
  - /path/to/route1.bat
  - /path/to/route2.bat
`)
		writeTempFile(tmpDir, "list2.yaml", `bat-file:
  - /path/to/route3.bat
  - /path/to/route4.bat
`)

		routerConfigPath := filepath.Join(tmpDir, "router.yaml")
		Expect(os.WriteFile(routerConfigPath, []byte(`keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - list1.yaml
      - list2.yaml
      - /path/to/direct.bat
`), 0644)).To(Succeed())

		Expect(config.LoadConfig(routerConfigPath)).To(Succeed())
		Expect(config.Cfg.Routes).To(HaveLen(1))

		Expect(config.Cfg.Routes[0].BatFile).To(Equal([]string{
			"/path/to/route1.bat",
			"/path/to/route2.bat",
			"/path/to/route3.bat",
			"/path/to/route4.bat",
			"/path/to/direct.bat",
		}))
	})
})
