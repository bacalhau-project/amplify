package config

import "time"

type Job struct {
	ID            string        `yaml:"id"`
	Type          string        `yaml:"type"`
	InternalJobID string        `yaml:"internal_job_id"`
	Image         string        `yaml:"image"`
	Entrypoint    []string      `yaml:"entrypoint"`
	Timeout       time.Duration `yaml:"timeout"`
	Memory        string        `yaml:"memory"`
	CPU           string        `yaml:"cpu"`
}
