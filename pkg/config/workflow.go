package config

type Workflow struct {
	Name string        `yaml:"name"`
	Jobs []WorkflowJob `yaml:"jobs"`
}

type WorkflowJob struct {
	Name string `yaml:"name"`
}
