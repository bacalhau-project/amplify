// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/workflow"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog/log"
)

type TaskFactory struct {
	jf   job.JobFactory
	wf   workflow.WorkflowFactory
	ng   ipldformat.NodeGetter
	exec executor.Executor
}

// NewTaskFactory creates a factory that makes it easier to create tasks for the
// workers
func NewTaskFactory(appContext cli.AppContext) (*TaskFactory, error) {
	// Config
	conf, err := config.GetConfig(appContext.Config.ConfigPath)
	if err != nil {
		return nil, err
	}

	// Job Factory
	jobFactory := job.NewJobFactory(*conf)

	// Workflow factory
	workflowFactory := workflow.NewWorkflowFactory(*conf)

	tf := TaskFactory{
		jf:   jobFactory,
		wf:   workflowFactory,
		ng:   appContext.NodeProvider,
		exec: appContext.Executor,
	}
	return &tf, nil
}

// TODO: This is horrible
func (f *TaskFactory) CreateWorkflowTask(ctx context.Context, name string, cid string) (queue.Callable, error) {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Info().Msgf("Running workflow %s", name)
		comp, err := f.buildComposite(ctx, cid)
		if err != nil {
			return err
		}
		workflow, err := f.wf.GetWorkflow(name)
		if err != nil {
			return err
		}
		for _, step := range workflow.Jobs {
			log.Ctx(ctx).Info().Msgf("Running job %s", step.Name)
			err = job.SingleJob{
				Executor: f.exec,
				Renderer: &f.jf,
			}.Run(ctx, step.Name, comp)
			if err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func (f *TaskFactory) CreateJobTask(ctx context.Context, name string, cid string) (queue.Callable, error) {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Info().Msgf("Running job %s", name)
		comp, err := f.buildComposite(ctx, cid)
		if err != nil {
			return err
		}
		j := job.SingleJob{
			Renderer: &f.jf,
			Executor: f.exec,
		}
		return j.Run(ctx, name, comp)
	}, nil
}

// Create a composite for the given CID
func (f *TaskFactory) buildComposite(ctx context.Context, c string) (*composite.Composite, error) {
	comp, err := composite.NewComposite(ctx, f.ng, cid.MustParse(c))
	if err != nil {
		return nil, err
	}
	return comp, nil
}
