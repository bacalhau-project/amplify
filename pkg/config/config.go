// Package config models the configuration file for Amplify
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	return &config, nil
}
