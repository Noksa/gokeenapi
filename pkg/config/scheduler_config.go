package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// SchedulerConfig represents scheduler configuration structure
type SchedulerConfig struct {
	// Tasks contains list of scheduled tasks to execute
	Tasks []ScheduledTask `yaml:"tasks"`
}

// ScheduledTask defines a single scheduled task
type ScheduledTask struct {
	// Name is a descriptive name for the task
	Name string `yaml:"name"`
	// Commands is the list of gokeenapi commands to execute sequentially (e.g., ["delete-routes", "add-routes"])
	Commands []string `yaml:"commands"`
	// Configs contains paths to config files to use for this task
	Configs []string `yaml:"configs"`
	// Interval specifies execution interval (e.g., "3h", "30m", "1h30m")
	Interval string `yaml:"interval,omitempty"`
	// Times specifies fixed execution times in 24h format (e.g., ["06:00", "12:00", "18:00"])
	Times []string `yaml:"times,omitempty"`
	// Retry specifies number of retry attempts on failure (default: 0)
	Retry int `yaml:"retry,omitempty"`
	// RetryDelay specifies delay between retries (e.g., "30s", "1m", default: "1m")
	RetryDelay string `yaml:"retryDelay,omitempty"`
	// Strategy defines execution strategy: "parallel" for concurrent execution across all configs, default is sequential
	Strategy string `yaml:"strategy,omitempty"`
}

// LoadSchedulerConfig loads scheduler configuration from YAML file
func LoadSchedulerConfig(configPath string) (SchedulerConfig, error) {
	if configPath == "" {
		return SchedulerConfig{}, errors.New("scheduler config path is empty")
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return SchedulerConfig{}, err
	}
	var cfg SchedulerConfig
	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		return SchedulerConfig{}, err
	}
	return cfg, nil
}
