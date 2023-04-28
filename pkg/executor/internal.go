package executor

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var (
	ErrInternalJobNotFound = fmt.Errorf("internal job not found")
	ErrOnlyOneInput        = fmt.Errorf("must only be one input")
	ErrFailJobError        = fmt.Errorf("this is a test job that always fails")
)

type InternalJob interface {
	Execute(context.Context) (Result, error)
}

func NewInternalExecutor() Executor {
	return &internalExecutor{}
}

type internalExecutor struct {
}

func (*internalExecutor) Execute(ctx context.Context, job config.Job, i interface{}) (Result, error) {
	return i.(InternalJob).Execute(ctx)
}

func (*internalExecutor) Render(job config.Job, inputs []ExecutorIOSpec, outputs []ExecutorIOSpec) (interface{}, error) {
	switch job.InternalJobID {
	case "root-job":
		return &rootJob{
			inputs: inputs,
		}, nil
	case "fail-job":
		return nil, ErrFailJobError
	default:
		return &missingJob{}, nil
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
			Status: model.JobStateError.String(),
		}, ErrOnlyOneInput
	}
	return Result{
		ID:     "internal",
		CID:    j.inputs[0].Ref,
		Status: model.JobStateCompleted.String(),
	}, nil
}

type missingJob struct{}

func (j *missingJob) Execute(ctx context.Context) (Result, error) {
	return Result{
		ID:     "internal",
		StdErr: ErrInternalJobNotFound.Error(),
		Status: model.JobStateError.String(),
	}, ErrInternalJobNotFound
}
