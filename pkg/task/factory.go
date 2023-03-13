// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/workflow"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

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
func (f *TaskFactory) CreateWorkflowTask(ctx context.Context, name string, cid string) (*dag.Node[*composite.Composite], error) {
	log.Ctx(ctx).Info().Msgf("Running workflow %s", name)
	workflow, err := f.wf.GetWorkflow(name)
	if err != nil {
		return nil, err
	}
	step := workflow.Jobs[0]
	dag := dag.NewNode(func(ctx context.Context, comp *composite.Composite) *composite.Composite {
		log.Ctx(ctx).Info().Str("step", step.Name).Msg("Running job")
		comp, err := f.buildComposite(ctx, cid)
		if err != nil {
			log.Fatal().Err(err).Msg("Error executing job")
		}
		j := f.render(step.Name, comp)
		r, err := f.exec.Execute(ctx, j)
		if err != nil {
			log.Fatal().Err(err).Msg("Error executing job")
		}
		comp.SetResult(r)
		newComp, err := composite.NewComposite(ctx, f.ng, comp.Result().CID)
		if err != nil {
			log.Fatal().Err(err).Msg("Error executing job")
		}
		return newComp
	}, nil)
	for _, step := range workflow.Jobs[1:] {
		dag.AddChild(func(ctx context.Context, comp *composite.Composite) *composite.Composite {
			log.Ctx(ctx).Info().Str("step", step.Name).Msg("Running job")
			if comp == nil {
				panic("composite is nil")
			}
			j := f.render(step.Name, comp)
			r, err := f.exec.Execute(ctx, j)
			if err != nil {
				log.Fatal().Err(err).Msg("Error executing job")
			}
			comp.SetResult(r)
			newComp, err := composite.NewComposite(ctx, f.ng, comp.Result().CID)
			if err != nil {
				log.Fatal().Err(err).Msg("Error executing job")
			}
			return newComp
		})
	}
	return dag, nil
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

func (f *TaskFactory) render(name string, comp *composite.Composite) interface{} {
	job, err := f.jf.GetJob(name)
	if err != nil {
		panic(err)
	}

	var j = model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}

	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: job.Image,
			// TODO: There's a lot going on here, and we should encapsulate it in code/container.
			Entrypoint: job.Entrypoint,
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          job.Outputs.Path,
			},
		},
		Annotations: []string{amplifyAnnotation},
		NodeSelectors: []model.LabelSelectorRequirement{
			{
				Key:      "owner",
				Operator: selection.Equals,
				Values:   []string{"bacalhau"},
			},
		},
	}

	// The root node in the composite is the original data
	rootIntput := model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           comp.Node().Cid().String(), // assume root node is root cid
		Path:          "/inputs",
	}
	j.Spec.Inputs = append(j.Spec.Inputs, rootIntput)

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}
