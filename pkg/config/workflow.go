package config

type WorkflowType string

const (
	WorkflowTypeMap    WorkflowType = "map"
	WorkflowTypeSingle WorkflowType = "single"
)

type Workflow struct {
	Name string        `yaml:"name"`
	Jobs []WorkflowJob `yaml:"jobs"`
}

type WorkflowJob struct {
	Name string       `yaml:"name"`
	Type WorkflowType `yaml:"type"`
}
