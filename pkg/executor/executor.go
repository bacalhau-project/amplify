// Package executor provides an abstraction of a job executor.
package executor

import (
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/ipfs/go-cid"
	"golang.org/x/net/context"
)

// Executor abstracts the execution of a job
type Executor interface {
	Execute(context.Context, interface{}) (Result, error)
	Render(config.Job, []string) interface{}
}

// Result is an Amplify abstraction of a job result
type Result struct {
	ID     string
	CID    cid.Cid
	StdErr string
	StdOut string
}
