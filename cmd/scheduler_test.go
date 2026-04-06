package cmd

import (
	"os"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scheduler", func() {
	It("should create command with correct attributes", func() {
		cmd := newSchedulerCmd()

		Expect(cmd.Use).To(Equal("scheduler"))
		Expect(cmd.Aliases).To(ContainElement("schedule"))
		Expect(cmd.Aliases).To(ContainElement("sched"))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	Describe("validateTask", func() {
		It("should accept valid task", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should reject missing name", func() {
			task := config.ScheduledTask{
				Name: "", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("should reject missing commands", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one command is required"))
		})

		It("should reject missing configs", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one config is required"))
		})

		It("should reject missing interval and times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either interval or times must be specified"))
		})

		It("should reject both interval and times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Times: []string{"12:00"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mutually exclusive"))
		})

		It("should reject invalid interval", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "invalid",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid interval format"))
		})

		It("should reject too short interval", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "500ms",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interval must be at least 1 second"))
		})

		It("should reject invalid time format", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Times: []string{"25:00"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid time format"))
		})

		It("should accept valid times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Times: []string{"06:00", "12:00", "18:00"},
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should accept multiple commands", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"delete-routes", "add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should accept valid retry config", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
				Retry: 3, RetryDelay: "30s",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should reject negative retry", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Retry: -1,
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retry must be >= 0"))
		})

		It("should reject invalid retry delay", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", RetryDelay: "invalid",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid retryDelay format"))
		})

		It("should reject too short retry delay", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []string{"add-routes"},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", RetryDelay: "500ms",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retryDelay must be at least 1 second"))
		})

		Context("strategy validation", func() {
			It("should accept sequential", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: []string{"add-routes"},
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "sequential",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should accept parallel", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: []string{"add-routes"},
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "parallel",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should accept empty strategy", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: []string{"add-routes"},
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should reject invalid strategy", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: []string{"add-routes"},
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "invalid",
				}
				err := validateTask(task)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid strategy"))
				Expect(err.Error()).To(ContainSubstring("sequential"))
				Expect(err.Error()).To(ContainSubstring("parallel"))
			})
		})
	})

	Describe("getNextRunTime", func() {
		It("should return future time for a future time string", func() {
			now := time.Now()
			futureTime := now.Add(2 * time.Hour).Format("15:04")

			nextRun := getNextRunTime([]string{futureTime})
			Expect(nextRun).To(BeTemporally(">", now))
			Expect(nextRun).To(BeTemporally("<", now.Add(3*time.Hour)))
		})

		It("should return next day for a past time string", func() {
			now := time.Now()
			pastTime := now.Add(-2 * time.Hour).Format("15:04")

			nextRun := getNextRunTime([]string{pastTime})
			Expect(nextRun).To(BeTemporally(">", now.Add(20*time.Hour)))
			Expect(nextRun).To(BeTemporally("<", now.Add(26*time.Hour)))
		})

		It("should pick the nearest future time from multiple", func() {
			times := []string{"10:00", "14:00", "18:00"}
			nextRun := getNextRunTime(times)
			now := time.Now()

			Expect(nextRun).To(BeTemporally(">", now.Add(-1*time.Minute)))
			Expect(times).To(ContainElement(nextRun.Format("15:04")))
		})

		It("should handle all past times", func() {
			times := []string{"01:00", "02:00", "03:00"}
			nextRun := getNextRunTime(times)

			Expect(times).To(ContainElement(nextRun.Format("15:04")))
			Expect(nextRun).To(BeTemporally(">", time.Now().Add(-1*time.Minute)))
		})
	})

	Describe("LoadSchedulerConfig", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "scheduler-test-*")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = os.RemoveAll(tmpDir)
			})
		})

		It("should load valid config", func() {
			configPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Test task"
    commands:
      - add-routes
    configs:
      - /path/to/config.yaml
    interval: "1h"
`)
			cfg, err := config.LoadSchedulerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Tasks).To(HaveLen(1))
			Expect(cfg.Tasks[0].Name).To(Equal("Test task"))
			Expect(cfg.Tasks[0].Commands).To(Equal([]string{"add-routes"}))
			Expect(cfg.Tasks[0].Interval).To(Equal("1h"))
		})

		It("should load config with multiple commands", func() {
			configPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Test task"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/config.yaml
    times:
      - "02:00"
`)
			cfg, err := config.LoadSchedulerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Tasks).To(HaveLen(1))
			Expect(cfg.Tasks[0].Commands).To(Equal([]string{"delete-routes", "add-routes"}))
			Expect(cfg.Tasks[0].Times).To(Equal([]string{"02:00"}))
		})

		It("should fail on empty path", func() {
			cfg, err := config.LoadSchedulerConfig("")
			Expect(err).To(HaveOccurred())
			Expect(cfg.Tasks).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should fail on non-existent file", func() {
			cfg, err := config.LoadSchedulerConfig("/nonexistent/scheduler.yaml")
			Expect(err).To(HaveOccurred())
			Expect(cfg.Tasks).To(BeEmpty())
		})

		It("should fail on invalid YAML", func() {
			configPath := writeTempFile(tmpDir, "invalid.yaml", "invalid: yaml: content:")

			cfg, err := config.LoadSchedulerConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(cfg.Tasks).To(BeEmpty())
		})
	})
})
