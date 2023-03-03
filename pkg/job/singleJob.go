package job

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/executor"
)

// SingleJob executes a single job for a given composite
type SingleJob struct {
	Executor executor.Executor
	Renderer Renderer
}

func (sj SingleJob) Run(ctx context.Context, jobName string, c *composite.Composite) error {
	j := sj.Renderer.Render(jobName, c)
	r, err := sj.Executor.Execute(ctx, j)
	if err != nil {
		return err
	}
	c.SetResult(r)
	return nil
}
