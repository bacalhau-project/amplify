// Package job provides an abstraction of a job that runs on Bacalhau.
//
// A job is part of an Amplify workflow that performs some kind of
// data related task. It's unit of work is a Bacalhau job.
package job

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/composite"
)

// Renderer abstracts the rendering of a job
type Renderer interface {
	Render(string, *composite.Composite) interface{}
}

// Runner abstracts the running of a job
type Runner interface {
	Run(context.Context, string, *composite.Composite) error
}
