package config

type Job struct {
	ID         string   `yaml:"id"`
	Image      string   `yaml:"image"`
	Entrypoint []string `yaml:"entrypoint"`
}
