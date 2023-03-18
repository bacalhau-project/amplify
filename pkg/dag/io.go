package dag

type IOSpec interface {
	NodeID() string
	ID() string
	CID() string
	Path() string
	IsRoot() bool
	SetExecutionInfo(ExecutionInfo)
	ExecutionInfo() ExecutionInfo
}

// IOSpec is a generic input/output specification for a DAG
type ioSpec struct {
	nodeID   string // Reference to which node this spec is attached to. E.g. a step ID
	id       string // Reference to the IO ID. E.g. the ID of an input
	value    string // Value of the input/output, if applicable. E.g. a CID
	path     string // E.g. a path
	root     bool   // If this input represents a root input
	execInfo ExecutionInfo
}

type ExecutionInfo struct {
	ID     string // External ID of the job
	Stdout string // Stdout of the job
	Stderr string // Stderr of the job
	Status string // Status of the job
}

func NewIOSpec(nodeID, id, value, path string, root bool) IOSpec {
	return &ioSpec{
		nodeID: nodeID,
		id:     id,
		value:  value,
		path:   path,
		root:   root,
	}
}

func (i ioSpec) NodeID() string {
	return i.nodeID
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

func (i ioSpec) ExecutionInfo() ExecutionInfo {
	return i.execInfo
}

func (i *ioSpec) SetExecutionInfo(e ExecutionInfo) {
	i.execInfo = e
}
