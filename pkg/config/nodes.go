package config

type Node struct {
	ID      string       `yaml:"id"`
	JobID   string       `yaml:"job_id"`
	Inputs  []NodeInput  `yaml:"inputs"`
	Outputs []NodeOutput `yaml:"outputs"`
}

type NodeInput struct {
	Root     bool   `yaml:"root"`
	StepID   string `yaml:"step_id"`
	OutputId string `yaml:"output_id"`
	Path     string `yaml:"path"`
}

type NodeOutput struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type WorkflowOptions struct {
	DisableDerivative bool   `yaml:"disable_derivative"`
	DerivativeJobName string `yaml:"derivative_job_name"`
}
