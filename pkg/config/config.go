// Package config models the configuration file for Amplify
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Jobs      []Job           `yaml:"jobs"`
	Workflows []Workflow      `yaml:"workflows"`
	Workflow  WorkflowOptions `yaml:"workflow"`
}

func GetConfig(path string) (*Config, error) {
	// Load yaml file from bundle
	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
