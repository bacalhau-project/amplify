package dag

// IOSpec is a generic input/output specification for a DAG
type IOSpec interface {
	NodeName() string // The node ID from the graph spec (not the internal node ID)
	ID() string       // The node's input ID from the graph spec
	CID() string
	Context() string
	Path() string
	IsRoot() bool
}

type ioSpec struct {
	nodeName string // Within a job, we don't have the graph, so we need a way to check the node
	id       string // Reference to the IO ID. E.g. the ID of an input
	value    string // Value of the input/output, if applicable. E.g. a CID
	path     string // E.g. a path
	root     bool   // If this input represents a root input
	context  string // The context of the input/output, e.g. stdout
}

type ExecutionInfo struct {
	ID     string // External ID of the job
	Stdout string // Stdout of the job
	Stderr string // Stderr of the job
	Status string // Status of the job
}

func NewIOSpec(name, id, value, path string, root bool, context string) IOSpec {
	return &ioSpec{
		nodeName: name,
		id:       id,
		value:    value,
		path:     path,
		root:     root,
		context:  context,
	}
}

func (i ioSpec) NodeName() string {
	return i.nodeName
}

// CID is an alias for Value
func (i ioSpec) CID() string {
	return i.value
}

func (i ioSpec) Path() string {
	return i.path
}

func (i ioSpec) ID() string {
	return i.id
}

func (i ioSpec) IsRoot() bool {
	return i.root
}

func (i ioSpec) Context() string {
	return i.context
}
