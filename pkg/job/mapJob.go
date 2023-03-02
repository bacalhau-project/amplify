package job

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/executor"
)

// MapJob creates multiple jobs for all leaves in a composite and records
// the result in the composite.
type MapJob struct {
	Executor executor.Executor
	Renderer Renderer
}

func (mj MapJob) Run(ctx context.Context, jobName string, comp *composite.Composite) error {
	var recurseCompositefunc func(context.Context, *composite.Composite) error
	recurseCompositefunc = func(ctx context.Context, c *composite.Composite) error {
		// Only run on leaf nodes. I.e. if it has children, don't run.
		if len(c.Children()) == 0 {
			j := mj.Renderer.Render(jobName, c)
			r, err := mj.Executor.Execute(ctx, j)
			if err != nil {
				return err
			}
			c.SetResult(r)
		}
		for _, child := range c.Children() {
			recurseCompositefunc(ctx, child)
		}
		return nil
	}
	if err := recurseCompositefunc(ctx, comp); err != nil {
		return err
	}
	return nil
}
