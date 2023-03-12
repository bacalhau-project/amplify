package config

type Job struct {
	Name       string   `yaml:"name"`
	Image      string   `yaml:"image"`
	Entrypoint []string `yaml:"entrypoint"`
	Inputs     Storage  `yaml:"inputs"`
	Outputs    Storage  `yaml:"outputs"`
}

type Storage struct {
	Path string `yaml:"path"`
}
