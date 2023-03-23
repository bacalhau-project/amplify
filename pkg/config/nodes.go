package config

type Node struct {
	ID      string       `yaml:"id"`
	JobID   string       `yaml:"job_id"`
	Inputs  []NodeInput  `yaml:"inputs"`
	Outputs []NodeOutput `yaml:"outputs"`
}

type NodeInput struct {
	Root      bool   `yaml:"root"`
	NodeID    string `yaml:"node_id"`
	OutputID  string `yaml:"output_id"`
	Path      string `yaml:"path"`
	Predicate string `yaml:"predicate"`
}

type NodeOutput struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}
