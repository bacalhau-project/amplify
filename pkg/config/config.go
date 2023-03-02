// Package config models the configuration file for Amplify
package config

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Jobs []Job `yaml:"jobs"`
}

func GetConfig(path string) (*Config, error) {
	// Load yaml file from bundle
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
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
