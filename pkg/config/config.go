// Package config models the configuration file for Amplify
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultTimeout = 10 * time.Minute
	DefaultCPU     = "1"
	DefaultMemory  = "1Gi"
)

type Config struct {
	Jobs  []Job  `yaml:"jobs"`
	Graph []Node `yaml:"graph"`
}

func GetConfig(path string) (*Config, error) {
	// Load yaml file from bundle
	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	defaultJobs := make([]Job, len(config.Jobs))
	for i, job := range config.Jobs {
		defaultJobs[i] = applyJobDefaults(job)
	}
	config.Jobs = defaultJobs
	return &config, nil
}

// ApplyDefaults applies the default values to the config
func applyJobDefaults(job Job) Job {
	if job.Timeout == 0 {
		job.Timeout = DefaultTimeout
	}
	if job.CPU == "" {
		job.CPU = DefaultCPU
	}
	if job.Memory == "" {
		job.Memory = DefaultMemory
	}
	return job
}
