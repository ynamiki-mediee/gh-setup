package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LabelConfig represents a single label definition.
type LabelConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

// MilestonesConfig represents milestone generation settings.
type MilestonesConfig struct {
	StartDate string `yaml:"startDate"`
	Weeks     int    `yaml:"weeks"`
	Timezone  string `yaml:"timezone"`
}

// Config is the top-level configuration loaded from .gh-setup.yml.
type Config struct {
	Milestones *MilestonesConfig `yaml:"milestones"`
	Labels     []LabelConfig     `yaml:"labels"`
}

const configFileName = ".gh-setup.yml"

// LoadConfig reads .gh-setup.yml from the current directory.
// It returns (nil, nil) if the file does not exist or is empty.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// An empty YAML document (e.g. only whitespace/comments) results in a
	// zero-value Config. Treat that the same as a missing file.
	if cfg.Milestones == nil && len(cfg.Labels) == 0 {
		return nil, nil
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	for i, l := range cfg.Labels {
		if l.Name == "" {
			return fmt.Errorf("labels[%d]: name is required", i)
		}
		if l.Color == "" {
			return fmt.Errorf("labels[%d]: color is required", i)
		}
	}

	if m := cfg.Milestones; m != nil {
		if m.StartDate == "" {
			return fmt.Errorf("milestones: startDate is required")
		}
		if _, err := time.Parse("2006-01-02", m.StartDate); err != nil {
			return fmt.Errorf("milestones: invalid startDate %q: must be YYYY-MM-DD", m.StartDate)
		}
		if m.Weeks <= 0 {
			return fmt.Errorf("milestones: weeks must be a positive integer")
		}
		if m.Timezone != "" {
			if _, err := time.LoadLocation(m.Timezone); err != nil {
				return fmt.Errorf("milestones: invalid timezone %q: %w", m.Timezone, err)
			}
		}
	}

	return nil
}
