package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/enriquebris/goconcurrentqueue"
	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/spf13/cobra"
)

func newSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdScheduler,
		Aliases: AliasesScheduler,
		Short:   "Run scheduled tasks from scheduler configuration",
		Long: `Run scheduled tasks based on scheduler configuration file.

The scheduler runs tasks at specified intervals or fixed times.
Each task executes one or more gokeenapi commands sequentially with one or more router configs.

The scheduler runs continuously until stopped (Ctrl+C). Tasks are executed sequentially
in a queue - if multiple tasks trigger at the same time, they will run one after another.
Commands within a task execute sequentially for each config.
If a command fails, remaining commands for that config are skipped.

Examples:
  # Run scheduler with config
  gokeenapi scheduler --config scheduler.yaml

  # Scheduler config example:
  tasks:
    - name: "Update routes every 3 hours"
      commands:
        - add-routes
      configs:
        - /path/to/router1.yaml
        - /path/to/router2.yaml
      interval: "3h"
    
    - name: "Refresh routes daily"
      commands:
        - delete-routes
        - add-routes
      configs:
        - /path/to/router1.yaml
      times:
        - "02:00"

Supported interval formats: "30m", "1h", "2h30m", "24h"
Supported time format: "HH:MM" (24-hour format)

Note: Use "interval" for periodic execution or "times" for fixed times.
Cannot use both in the same task.`,
		RunE: runScheduler,
	}
	return cmd
}

// runScheduler executes the scheduler command
func runScheduler(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	schedulerCfg, err := config.LoadSchedulerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load scheduler config: %w", err)
	}

	if len(schedulerCfg.Tasks) == 0 {
		return fmt.Errorf("no tasks defined in scheduler config")
	}

	gokeenlog.Info(color.GreenString("🕐 Scheduler started"))
	gokeenlog.InfoSubStepf("Tasks: %v", color.CyanString("%v", len(schedulerCfg.Tasks)))

	ctx := cmd.Context()

	// Create FIFO queue for sequential task execution
	queue := goconcurrentqueue.NewFIFO()

	// Start task executor (single worker for sequential execution)
	go func() {
		time.Sleep(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				item, err := queue.DequeueOrWaitForNextElement()
				if err == nil {
					if task, ok := item.(config.ScheduledTask); ok {
						executeTask(task)
					}
				}
			}
		}
	}()

	for _, task := range schedulerCfg.Tasks {
		if err := validateTask(task); err != nil {
			return fmt.Errorf("task %q validation failed: %w", task.Name, err)
		}
		go runTask(ctx, task, queue)
	}

	<-ctx.Done()
	gokeenlog.Info(color.YellowString("🛑 Scheduler stopped"))
	return nil
}

// validateTask validates task configuration
func validateTask(task config.ScheduledTask) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if len(task.Commands) == 0 {
		return fmt.Errorf("at least one command is required")
	}
	if len(task.Configs) == 0 {
		return fmt.Errorf("at least one config is required")
	}
	if task.Interval == "" && len(task.Times) == 0 {
		return fmt.Errorf("either interval or times must be specified")
	}
	if task.Interval != "" && len(task.Times) > 0 {
		return fmt.Errorf("interval and times are mutually exclusive")
	}
	if task.Interval != "" {
		if _, err := time.ParseDuration(task.Interval); err != nil {
			return fmt.Errorf("invalid interval format: %w", err)
		}
	}
	for _, t := range task.Times {
		if _, err := time.Parse("15:04", t); err != nil {
			return fmt.Errorf("invalid time format %q (use HH:MM): %w", t, err)
		}
	}
	return nil
}

// runTask runs a scheduled task
func runTask(ctx context.Context, task config.ScheduledTask, queue goconcurrentqueue.Queue) {
	if task.Interval != "" {
		runIntervalTask(ctx, task, queue)
	} else {
		runTimedTask(ctx, task, queue)
	}
}

// runIntervalTask runs task at specified intervals
func runIntervalTask(ctx context.Context, task config.ScheduledTask, queue goconcurrentqueue.Queue) {
	interval, _ := time.ParseDuration(task.Interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	gokeenlog.InfoSubStepf("Task '%v': running every %s", color.CyanString(task.Name), color.BlueString("%v", interval))

	_ = queue.Enqueue(task)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = queue.Enqueue(task)
		}
	}
}

// runTimedTask runs task at specified times
func runTimedTask(ctx context.Context, task config.ScheduledTask, queue goconcurrentqueue.Queue) {
	gokeenlog.InfoSubStepf("Task '%v': running at %v", color.CyanString(task.Name), color.BlueString("%v", task.Times))

	for {
		nextRun := getNextRunTime(task.Times)
		waitDuration := time.Until(nextRun)

		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
			_ = queue.Enqueue(task)
		}
	}
}

// getNextRunTime calculates next execution time from times list
func getNextRunTime(times []string) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var nextRun time.Time
	var earliestTime time.Time

	for _, t := range times {
		parsed, _ := time.Parse("15:04", t)
		runTime := today.Add(time.Hour*time.Duration(parsed.Hour()) + time.Minute*time.Duration(parsed.Minute()))

		// Track earliest time for tomorrow
		if earliestTime.IsZero() || runTime.Before(earliestTime) {
			earliestTime = runTime
		}

		// Find next run today
		if runTime.After(now) && (nextRun.IsZero() || runTime.Before(nextRun)) {
			nextRun = runTime
		}
	}

	// If no time left today, use earliest time tomorrow
	if nextRun.IsZero() {
		nextRun = earliestTime.Add(24 * time.Hour)
	}

	return nextRun
}

// executeTask executes a single task
func executeTask(task config.ScheduledTask) {
	gokeenlog.Infof("▶ Executing task: %v", color.BlueString(task.Name))

	for _, configPath := range task.Configs {
		gokeenlog.InfoSubStepf("Config: %s", color.CyanString(configPath))

		for _, command := range task.Commands {
			executable, err := os.Executable()
			if err != nil {
				gokeenlog.InfoSubStepf("%s Failed to get executable path: %v", color.RedString("✗"), err)
				continue
			}

			b := &strings.Builder{}
			err = gokeenspinner.WrapWithSpinner(fmt.Sprintf("Executing %s", color.BlueString(command)), func() error {
				cmd := exec.Command(executable, command, "--config", configPath)
				cmd.Stdout = b
				cmd.Stderr = b
				return cmd.Run()
			})

			if err != nil {
				gokeenlog.InfoSubStepf("Error: %v, %v", color.RedString(err.Error()), color.RedString(b.String()))
				break
			}
		}
	}
}
