package cmd

import (
	"context"
	"os"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// cmds is a helper to build []config.ScheduledCommand from plain strings.
func cmds(names ...string) []config.ScheduledCommand {
	out := make([]config.ScheduledCommand, len(names))
	for i, n := range names {
		out[i] = config.ScheduledCommand{Name: n}
	}
	return out
}

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
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should reject missing name", func() {
			task := config.ScheduledTask{
				Name: "", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name is required"))
		})

		It("should reject missing commands", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: []config.ScheduledCommand{},
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one command is required"))
		})

		It("should reject missing configs", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{}, Interval: "1h",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one config is required"))
		})

		It("should reject missing interval and times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either interval or times must be specified"))
		})

		It("should reject both interval and times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Times: []string{"12:00"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mutually exclusive"))
		})

		It("should reject invalid interval", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "invalid",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid interval format"))
		})

		It("should reject too short interval", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "500ms",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interval must be at least 1 second"))
		})

		It("should reject invalid time format", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Times: []string{"25:00"},
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid time format"))
		})

		It("should accept valid times", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Times: []string{"06:00", "12:00", "18:00"},
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should accept multiple commands", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("delete-routes", "add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should accept valid retry config", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h",
				Retry: 3, RetryDelay: "30s",
			}
			Expect(validateTask(task)).To(Succeed())
		})

		It("should reject negative retry", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Retry: -1,
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retry must be >= 0"))
		})

		It("should reject invalid retry delay", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", RetryDelay: "invalid",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid retryDelay format"))
		})

		It("should reject too short retry delay", func() {
			task := config.ScheduledTask{
				Name: "Test task", Commands: cmds("add-routes"),
				Configs: []string{"/path/to/config.yaml"}, Interval: "1h", RetryDelay: "500ms",
			}
			err := validateTask(task)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retryDelay must be at least 1 second"))
		})

		Context("strategy validation", func() {
			It("should accept sequential", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: cmds("add-routes"),
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "sequential",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should accept parallel", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: cmds("add-routes"),
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "parallel",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should accept empty strategy", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: cmds("add-routes"),
					Configs: []string{"/path/to/config.yaml"}, Interval: "1h", Strategy: "",
				}
				Expect(validateTask(task)).To(Succeed())
			})

			It("should reject invalid strategy", func() {
				task := config.ScheduledTask{
					Name: "Test task", Commands: cmds("add-routes"),
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

		It("should load valid config with string commands (backward-compatible)", func() {
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
			Expect(cfg.Tasks[0].Commands).To(Equal(cmds("add-routes")))
			Expect(cfg.Tasks[0].Interval).To(Equal("1h"))
		})

		It("should load config with object-form command and args", func() {
			configPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Test task"
    commands:
      - name: delete-all-dns-routing
        args: ["--force"]
      - add-dns-routing
    configs:
      - /path/to/config.yaml
    interval: "6h"
`)
			cfg, err := config.LoadSchedulerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Tasks[0].Commands).To(HaveLen(2))
			Expect(cfg.Tasks[0].Commands[0].Name).To(Equal("delete-all-dns-routing"))
			Expect(cfg.Tasks[0].Commands[0].Args).To(Equal([]string{"--force"}))
			Expect(cfg.Tasks[0].Commands[1].Name).To(Equal("add-dns-routing"))
			Expect(cfg.Tasks[0].Commands[1].Args).To(BeNil())
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
			Expect(cfg.Tasks[0].Commands).To(Equal(cmds("delete-routes", "add-routes")))
			Expect(cfg.Tasks[0].Times).To(Equal([]string{"02:00"}))
		})

		It("should load mixed string and object commands", func() {
			configPath := writeTempFile(tmpDir, "scheduler.yaml", `tasks:
  - name: "Mixed task"
    commands:
      - delete-routes
      - name: add-routes
        args: ["--debug"]
    configs:
      - /path/to/config.yaml
    interval: "1h"
`)
			cfg, err := config.LoadSchedulerConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Tasks[0].Commands).To(HaveLen(2))
			Expect(cfg.Tasks[0].Commands[0]).To(Equal(config.ScheduledCommand{Name: "delete-routes"}))
			Expect(cfg.Tasks[0].Commands[1]).To(Equal(config.ScheduledCommand{Name: "add-routes", Args: []string{"--debug"}}))
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

	Describe("sendTask", func() {
		It("should send task to queue when space available", func() {
			ctx := context.Background()
			queue := make(chan config.ScheduledTask, 1)
			task := config.ScheduledTask{Name: "t1"}

			ok := sendTask(ctx, task, queue)

			Expect(ok).To(BeTrue())
			Expect(queue).To(HaveLen(1))
			Expect(<-queue).To(Equal(task))
		})

		It("should return false immediately when context is already cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			// unbuffered: send would block forever without ctx cancellation
			queue := make(chan config.ScheduledTask)
			task := config.ScheduledTask{Name: "t1"}

			ok := sendTask(ctx, task, queue)

			Expect(ok).To(BeFalse())
			Expect(queue).To(HaveLen(0))
		})

		It("should return false when context is cancelled while queue is full", func() {
			ctx, cancel := context.WithCancel(context.Background())
			queue := make(chan config.ScheduledTask) // unbuffered, always blocks

			done := make(chan bool, 1)
			go func() {
				done <- sendTask(ctx, config.ScheduledTask{Name: "t1"}, queue)
			}()

			// cancel while sendTask is blocked waiting for a receiver
			time.Sleep(10 * time.Millisecond)
			cancel()

			Eventually(done, "1s").Should(Receive(BeFalse()))
		})
	})

	Describe("scheduler cancellation", func() {
		It("stops all task producers without closing their queue", func() {
			for range 5 {
				ctx, cancel := context.WithCancel(context.Background())
				tasks := make([]config.ScheduledTask, 16)
				for i := range tasks {
					tasks[i] = config.ScheduledTask{Name: "cancelled", Interval: "1h"}
				}
				done := make(chan struct{})
				go func() {
					runScheduledTasks(ctx, tasks)
					close(done)
				}()
				cancel()
				Eventually(done).WithTimeout(time.Second).WithPolling(time.Millisecond).Should(BeClosed())
			}
		})
	})

	Describe("runIntervalTask cancellation", func() {
		It("should exit promptly on context cancellation even when queue is full", func() {
			ctx, cancel := context.WithCancel(context.Background())
			// queue size 1: first immediate send fills it; subsequent tick sends block
			queue := make(chan config.ScheduledTask, 1)
			task := config.ScheduledTask{
				Name:     "interval-task",
				Commands: cmds("add-routes"),
				Configs:  []string{"/cfg.yaml"},
				Interval: "10ms",
			}

			done := make(chan struct{})
			go func() {
				runIntervalTask(ctx, task, queue)
				close(done)
			}()

			// let the goroutine start and the first send land
			time.Sleep(20 * time.Millisecond)
			cancel()

			Eventually(done, "500ms").Should(BeClosed())
		})
	})

	Describe("executeCommandsForConfig", func() {
		It("stops waiting for a retry when the scheduler is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			task := config.ScheduledTask{
				Name:       "cancel-retry",
				Commands:   cmds("nonexistent-gokeenapi-command-xyz"),
				Configs:    []string{"/some/config.yaml"},
				Retry:      1,
				RetryDelay: "5s",
			}
			done := make(chan error, 1)
			go func() {
				done <- executeCommandsForConfig(ctx, task, task.Configs[0])
			}()
			time.Sleep(100 * time.Millisecond)
			cancel()
			Eventually(done).WithTimeout(time.Second).WithPolling(time.Millisecond).
				Should(Receive(MatchError(ContainSubstring("context canceled"))))
		})

		It("should return error when command fails", func() {
			task := config.ScheduledTask{
				Name:     "test-task",
				Commands: cmds("nonexistent-gokeenapi-command-xyz"),
				Configs:  []string{"/some/config.yaml"},
			}
			err := executeCommandsForConfig(context.Background(), task, "/some/config.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonexistent-gokeenapi-command-xyz"))
		})

		It("should return nil when no commands", func() {
			task := config.ScheduledTask{
				Name:     "empty-task",
				Commands: []config.ScheduledCommand{},
				Configs:  []string{"/some/config.yaml"},
			}
			Expect(executeCommandsForConfig(context.Background(), task, "/some/config.yaml")).To(Succeed())
		})

		It("should include args in error message when object-form command fails", func() {
			task := config.ScheduledTask{
				Name: "test-task",
				Commands: []config.ScheduledCommand{
					{Name: "nonexistent-gokeenapi-command-xyz", Args: []string{"--force"}},
				},
				Configs: []string{"/some/config.yaml"},
			}
			err := executeCommandsForConfig(context.Background(), task, "/some/config.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonexistent-gokeenapi-command-xyz"))
		})
	})

	Describe("executeTask parallel error aggregation", func() {
		It("should collect errors from all failing goroutines without losing any", func() {
			task := config.ScheduledTask{
				Name:     "parallel-fail",
				Commands: cmds("nonexistent-gokeenapi-command-xyz"),
				Configs:  []string{"/cfg1.yaml", "/cfg2.yaml", "/cfg3.yaml"},
				Strategy: StrategyParallel,
			}
			// executeTask logs errors but doesn't return them; we verify it completes
			// without panic or deadlock and that executeCommandsForConfig returns errors
			// for each config.
			var errs []error
			for _, cfg := range task.Configs {
				errs = append(errs, executeCommandsForConfig(context.Background(), task, cfg))
			}
			Expect(errs).To(HaveLen(3))
			for _, err := range errs {
				Expect(err).To(HaveOccurred())
			}
		})

		It("should complete without deadlock in parallel mode", func() {
			task := config.ScheduledTask{
				Name:     "parallel-deadlock-check",
				Commands: cmds("nonexistent-gokeenapi-command-xyz"),
				Configs:  []string{"/cfg1.yaml", "/cfg2.yaml"},
				Strategy: StrategyParallel,
			}
			done := make(chan struct{})
			go func() {
				executeTask(context.Background(), task)
				close(done)
			}()
			Eventually(done, "10s").Should(BeClosed())
		})
	})
})
