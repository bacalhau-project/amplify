// Package executor provides an abstraction of a job executor.
package executor

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/ipfs/go-cid"
	"golang.org/x/net/context"
)

type ExecutorIOSpec struct {
	Name string
	Ref  string
	Path string
}

// Executor abstracts the execution of a job
type Executor interface {
	Execute(context.Context, interface{}) (Result, error)
	Render(config.Job, []ExecutorIOSpec, []ExecutorIOSpec) interface{}
}

// Result is an Amplify abstraction of a job result
type Result struct {
	ID     string
	CID    cid.Cid
	StdErr string
	StdOut string
	Status string
}
