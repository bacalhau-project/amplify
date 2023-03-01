package config

type StorageType string

const (
	StorageTypeSingle      StorageType = "single"
	StorageTypeIncremental StorageType = "incremental"
)

type Job struct {
	Name       string   `yaml:"name"`
	Image      string   `yaml:"image"`
	Entrypoint []string `yaml:"entrypoint"`
	Inputs     Storage  `yaml:"inputs"`
	Outputs    Storage  `yaml:"outputs"`
}

type Storage struct {
	Type StorageType `yaml:"type"`
	Path string      `yaml:"path"`
}
