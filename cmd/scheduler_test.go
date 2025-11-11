package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SchedulerTestSuite struct {
	suite.Suite
	tempDir string
}

func TestSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTestSuite))
}

func (s *SchedulerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "scheduler_test")
	s.Require().NoError(err)
}

func (s *SchedulerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func (s *SchedulerTestSuite) TestNewSchedulerCmd() {
	cmd := newSchedulerCmd()

	assert.Equal(s.T(), "scheduler", cmd.Use)
	assert.Contains(s.T(), cmd.Aliases, "schedule")
	assert.Contains(s.T(), cmd.Aliases, "sched")
	assert.NotEmpty(s.T(), cmd.Short)
	assert.NotNil(s.T(), cmd.RunE)
}

func (s *SchedulerTestSuite) TestValidateTask_Valid() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
	}

	err := validateTask(task)
	assert.NoError(s.T(), err)
}

func (s *SchedulerTestSuite) TestValidateTask_MissingName() {
	task := config.ScheduledTask{
		Name:     "",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "name is required")
}

func (s *SchedulerTestSuite) TestValidateTask_MissingCommands() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "at least one command is required")
}

func (s *SchedulerTestSuite) TestValidateTask_MissingConfigs() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{},
		Interval: "1h",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "at least one config is required")
}

func (s *SchedulerTestSuite) TestValidateTask_MissingIntervalAndTimes() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "either interval or times must be specified")
}

func (s *SchedulerTestSuite) TestValidateTask_BothIntervalAndTimes() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
		Times:    []string{"12:00"},
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "mutually exclusive")
}

func (s *SchedulerTestSuite) TestValidateTask_InvalidInterval() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "invalid",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid interval format")
}

func (s *SchedulerTestSuite) TestValidateTask_InvalidTimeFormat() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Times:    []string{"25:00"},
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid time format")
}

func (s *SchedulerTestSuite) TestValidateTask_ValidTimes() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Times:    []string{"06:00", "12:00", "18:00"},
	}

	err := validateTask(task)
	assert.NoError(s.T(), err)
}

func (s *SchedulerTestSuite) TestValidateTask_MultipleCommands() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"delete-routes", "add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
	}

	err := validateTask(task)
	assert.NoError(s.T(), err)
}

func (s *SchedulerTestSuite) TestValidateTask_WithRetry() {
	task := config.ScheduledTask{
		Name:       "Test task",
		Commands:   []string{"add-routes"},
		Configs:    []string{"/path/to/config.yaml"},
		Interval:   "1h",
		Retry:      3,
		RetryDelay: "30s",
	}

	err := validateTask(task)
	assert.NoError(s.T(), err)
}

func (s *SchedulerTestSuite) TestValidateTask_NegativeRetry() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "1h",
		Retry:    -1,
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "retry must be >= 0")
}

func (s *SchedulerTestSuite) TestValidateTask_InvalidRetryDelay() {
	task := config.ScheduledTask{
		Name:       "Test task",
		Commands:   []string{"add-routes"},
		Configs:    []string{"/path/to/config.yaml"},
		Interval:   "1h",
		RetryDelay: "invalid",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid retryDelay format")
}

func (s *SchedulerTestSuite) TestValidateTask_TooShortRetryDelay() {
	task := config.ScheduledTask{
		Name:       "Test task",
		Commands:   []string{"add-routes"},
		Configs:    []string{"/path/to/config.yaml"},
		Interval:   "1h",
		RetryDelay: "500ms",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "retryDelay must be at least 1 second")
}

func (s *SchedulerTestSuite) TestValidateTask_TooShortInterval() {
	task := config.ScheduledTask{
		Name:     "Test task",
		Commands: []string{"add-routes"},
		Configs:  []string{"/path/to/config.yaml"},
		Interval: "500ms",
	}

	err := validateTask(task)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "interval must be at least 1 second")
}

func (s *SchedulerTestSuite) TestGetNextRunTime_FutureTime() {
	now := time.Now()
	futureTime := now.Add(2 * time.Hour).Format("15:04")
	times := []string{futureTime}

	nextRun := getNextRunTime(times)

	assert.True(s.T(), nextRun.After(now))
	assert.True(s.T(), nextRun.Before(now.Add(3*time.Hour)))
}

func (s *SchedulerTestSuite) TestGetNextRunTime_PastTime() {
	now := time.Now()
	pastTime := now.Add(-2 * time.Hour).Format("15:04")
	times := []string{pastTime}

	nextRun := getNextRunTime(times)

	// Should be tomorrow
	assert.True(s.T(), nextRun.After(now.Add(20*time.Hour)))
	assert.True(s.T(), nextRun.Before(now.Add(26*time.Hour)))
}

func (s *SchedulerTestSuite) TestGetNextRunTime_MultipleTimes() {
	// Use fixed times that are definitely in the future relative to each other
	times := []string{"10:00", "14:00", "18:00"}

	nextRun := getNextRunTime(times)
	now := time.Now()

	// NextRun should be one of the specified times (today or tomorrow)
	assert.True(s.T(), nextRun.After(now.Add(-1*time.Minute)), "nextRun should be in the future")

	// Check that the time matches one of our specified times
	nextRunTime := nextRun.Format("15:04")
	assert.Contains(s.T(), times, nextRunTime, "nextRun should match one of the specified times")
}

func (s *SchedulerTestSuite) TestGetNextRunTime_AllPastTimes() {
	// Test with times that ensure we get earliest time for next day
	// Use times that are likely all in the past (early morning)
	times := []string{"01:00", "02:00", "03:00"}

	nextRun := getNextRunTime(times)

	// Should be one of the specified times
	nextRunTime := nextRun.Format("15:04")
	assert.Contains(s.T(), times, nextRunTime, "should pick one of the specified times")

	// Should be in the future
	assert.True(s.T(), nextRun.After(time.Now().Add(-1*time.Minute)), "should be in the future")
}

func (s *SchedulerTestSuite) TestLoadSchedulerConfig_Valid() {
	configPath := filepath.Join(s.tempDir, "scheduler.yaml")
	configContent := `tasks:
  - name: "Test task"
    commands:
      - add-routes
    configs:
      - /path/to/config.yaml
    interval: "1h"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.LoadSchedulerConfig(configPath)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), cfg.Tasks, 1)
	assert.Equal(s.T(), "Test task", cfg.Tasks[0].Name)
	assert.Equal(s.T(), []string{"add-routes"}, cfg.Tasks[0].Commands)
	assert.Equal(s.T(), "1h", cfg.Tasks[0].Interval)
}

func (s *SchedulerTestSuite) TestLoadSchedulerConfig_MultipleCommands() {
	configPath := filepath.Join(s.tempDir, "scheduler.yaml")
	configContent := `tasks:
  - name: "Test task"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/config.yaml
    times:
      - "02:00"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.LoadSchedulerConfig(configPath)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), cfg.Tasks, 1)
	assert.Equal(s.T(), []string{"delete-routes", "add-routes"}, cfg.Tasks[0].Commands)
	assert.Equal(s.T(), []string{"02:00"}, cfg.Tasks[0].Times)
}

func (s *SchedulerTestSuite) TestLoadSchedulerConfig_EmptyPath() {
	cfg, err := config.LoadSchedulerConfig("")
	assert.Error(s.T(), err)
	assert.Empty(s.T(), cfg.Tasks)
	assert.Contains(s.T(), err.Error(), "empty")
}

func (s *SchedulerTestSuite) TestLoadSchedulerConfig_NonExistent() {
	cfg, err := config.LoadSchedulerConfig("/nonexistent/scheduler.yaml")
	assert.Error(s.T(), err)
	assert.Empty(s.T(), cfg.Tasks)
}

func (s *SchedulerTestSuite) TestLoadSchedulerConfig_InvalidYAML() {
	configPath := filepath.Join(s.tempDir, "invalid.yaml")
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
	s.Require().NoError(err)

	cfg, err := config.LoadSchedulerConfig(configPath)
	assert.Error(s.T(), err)
	assert.Empty(s.T(), cfg.Tasks)
}
