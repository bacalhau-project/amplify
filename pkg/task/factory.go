// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"
	"errors"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/ipfs/go-cid"
	ipldformat "github.com/ipfs/go-ipld-format"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/selection"
)

const amplifyAnnotation = "amplify"

var ErrJobNotFound = errors.New("job not found")
var ErrWorkflowNotFound = errors.New("workflow not found")

type WorkflowJob struct {
	Name string
}

type Workflow struct {
	Name string
	Jobs []WorkflowJob
}

type TaskFactory struct {
	ng   ipldformat.NodeGetter
	exec executor.Executor
	conf config.Config
}

// NewTaskFactory creates a factory that makes it easier to create tasks for the
// workers
func NewTaskFactory(appContext cli.AppContext) (*TaskFactory, error) {
	// Config
	conf, err := config.GetConfig(appContext.Config.ConfigPath)
	if err != nil {
		return nil, err
	}

	tf := TaskFactory{
		ng:   appContext.NodeProvider,
		exec: appContext.Executor,
		conf: *conf,
	}
	return &tf, nil
}

// TODO: This is horrible
func (f *TaskFactory) CreateWorkflowTask(ctx context.Context, name string, cid string) (*dag.Node[*composite.Composite], error) {
	log.Ctx(ctx).Info().Msgf("Running workflow %s", name)
	workflow, err := f.GetWorkflow(name)
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
		newComp, err := composite.NewComposite(ctx, comp.Result().CID)
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
			newComp, err := composite.NewComposite(ctx, comp.Result().CID)
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
		comp, err := f.buildComposite(ctx, cid)
		if err != nil {
			log.Fatal().Err(err).Msg("Error executing job")
		}
		j := f.render(name, comp)
		r, err := f.exec.Execute(ctx, j)
		if err != nil {
			log.Fatal().Err(err).Msg("Error executing job")
		}
		comp.SetResult(r)
		return nil
	}, nil
}

// Create a composite for the given CID
func (f *TaskFactory) buildComposite(ctx context.Context, c string) (*composite.Composite, error) {
	comp, err := composite.NewComposite(ctx, cid.MustParse(c))
	if err != nil {
		return nil, err
	}
	return comp, nil
}

func (f *TaskFactory) render(name string, comp *composite.Composite) interface{} {
	job, err := f.GetJob(name)
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
		CID:           comp.Cid().String(), // assume root node is root cid
		Path:          "/inputs",
	}
	j.Spec.Inputs = append(j.Spec.Inputs, rootIntput)

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}

// GetJob gets a job config from a job factory
func (f *TaskFactory) GetJob(name string) (config.Job, error) {
	for _, job := range f.conf.Jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return config.Job{}, ErrJobNotFound
}

// JobNames returns all the names of the jobs in a job factory
func (f *TaskFactory) JobNames() []string {
	var names []string
	for _, job := range f.conf.Jobs {
		names = append(names, job.Name)
	}
	return names
}

func (f *TaskFactory) GetWorkflow(workflow string) (Workflow, error) {
	for _, w := range f.conf.Workflows {
		if w.Name == workflow {
			return f.createWorkflow(w)
		}
	}
	return Workflow{}, ErrWorkflowNotFound
}

func (f *TaskFactory) WorkflowNames() []string {
	var workflows []string
	for _, w := range f.conf.Workflows {
		workflows = append(workflows, w.Name)
	}
	return workflows
}

func (f *TaskFactory) createWorkflow(workflow config.Workflow) (Workflow, error) {
	w := Workflow{
		Name: workflow.Name,
	}
	for _, j := range workflow.Jobs {
		w.Jobs = append(w.Jobs, WorkflowJob{
			Name: j.Name,
		})
	}
	return w, nil
}
