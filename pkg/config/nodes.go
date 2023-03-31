package config

type Node struct {
	ID      string       `yaml:"id"`
	JobID   string       `yaml:"job_id"`
	Inputs  []NodeInput  `yaml:"inputs"`
	Outputs []NodeOutput `yaml:"outputs"`
}

func (n *Node) ApplyDefaults() {
	for i := range n.Inputs {
		if n.Inputs[i].OutputID == "" {
			n.Inputs[i].OutputID = "default"
		}
	}
	for i := range n.Outputs {
		if n.Outputs[i].ID == "" {
			n.Outputs[i].ID = "default"
		}
	}
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

func (n Node) IsRoot() bool {
	isRoot := false
	for _, i := range n.Inputs {
		if i.Root {
			isRoot = true
			break
		}
	}
	return isRoot
}
