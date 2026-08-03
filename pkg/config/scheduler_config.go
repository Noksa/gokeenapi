package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ScheduledCommand represents a single command with optional extra arguments.
// It can be specified as a plain string (backward-compatible) or as an object:
//
//	# string form
//	commands:
//	  - add-dns-routing
//
//	# object form (with extra flags)
//	commands:
//	  - name: delete-all-dns-routing
//	    args: ["--force"]
type ScheduledCommand struct {
	// Name is the gokeenapi command to run (e.g., "add-routes", "delete-all-dns-routing")
	Name string
	// Args contains extra flags/arguments appended after the command name (e.g., ["--force"])
	Args []string
}

// UnmarshalYAML implements custom unmarshaling so that a plain string
// and an object {name: ..., args: [...]} are both valid.
func (c *ScheduledCommand) UnmarshalYAML(node *yaml.Node) error {
	// Plain string form: "- add-routes"
	if node.Kind == yaml.ScalarNode {
		if node.Value == "" {
			return fmt.Errorf("command name cannot be empty")
		}
		c.Name = node.Value
		return nil
	}

	// Object form: "- name: delete-all-dns-routing\n  args: [--force]"
	type scheduledCommandAlias ScheduledCommand
	alias := (*scheduledCommandAlias)(c)
	if err := node.Decode(alias); err != nil {
		return err
	}
	if c.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	return nil
}

// SchedulerConfig represents scheduler configuration structure
type SchedulerConfig struct {
	// Tasks contains list of scheduled tasks to execute
	Tasks []ScheduledTask `yaml:"tasks"`
}

// ScheduledTask defines a single scheduled task
type ScheduledTask struct {
	// Name is a descriptive name for the task
	Name string `yaml:"name"`
	// Commands is the list of gokeenapi commands to execute sequentially.
	// Each entry can be a plain string or an object with name and args fields.
	Commands []ScheduledCommand `yaml:"commands"`
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
