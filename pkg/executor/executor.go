// Package executor provides an abstraction of a job executor.
package executor

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/config"
)

type ExecutorIOSpec struct {
	Name string
	Ref  string
	Path string
}

// Executor abstracts the execution of a job
type Executor interface {
	Execute(context.Context, config.Job, interface{}) (Result, error)
	Render(config.Job, []ExecutorIOSpec, []ExecutorIOSpec) (interface{}, error)
}

// Result is an Amplify abstraction of a job result
type Result struct {
	ID     string
	CID    string
	StdErr string
	StdOut string
	Status string
}
