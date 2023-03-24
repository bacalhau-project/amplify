package executor

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/ipfs/go-cid"
)

var ErrInternalJobNotFound = fmt.Errorf("internal job not found")
var ErrOnlyOneInput = fmt.Errorf("must only be one input")

type InternalJob interface {
	Execute(context.Context) (Result, error)
}

func NewInternalExecutor() Executor {
	return &internalExecutor{}
}

type internalExecutor struct {
}

func (*internalExecutor) Execute(ctx context.Context, i interface{}) (Result, error) {
	return i.(InternalJob).Execute(ctx)
}

func (*internalExecutor) Render(job config.Job, inputs []ExecutorIOSpec, outputs []ExecutorIOSpec) interface{} {
	switch job.InternalJobID {
	case "root-job":
		return &rootJob{
			inputs: inputs,
		}
	default:
		return &missingJob{}
	}
}

type rootJob struct {
	inputs []ExecutorIOSpec
}

func (j *rootJob) Execute(ctx context.Context) (Result, error) {
	if len(j.inputs) != 1 {
		return Result{
			ID:     "internal",
			StdErr: ErrOnlyOneInput.Error(),
			Status: "error",
		}, ErrOnlyOneInput
	}
	cid, err := cid.Parse(j.inputs[0].Ref)
	if err != nil {
		return Result{
			ID:     "internal",
			StdErr: err.Error(),
			Status: "error",
		}, err
	}
	return Result{
		ID:     "internal",
		CID:    cid,
		Status: "Completed",
	}, nil
}

type missingJob struct{}

func (j *missingJob) Execute(ctx context.Context) (Result, error) {
	return Result{
		ID:     "internal",
		StdErr: ErrInternalJobNotFound.Error(),
		Status: "error",
	}, ErrInternalJobNotFound
}
