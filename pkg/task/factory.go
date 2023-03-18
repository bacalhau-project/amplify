// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"
	"errors"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/rs/zerolog/log"
)

var ErrJobNotFound = errors.New("job not found")
var ErrWorkflowNotFound = errors.New("workflow not found")
var ErrWorkflowNoJobs = errors.New("workflow has no jobs")
var ErrEmptyWorkflows = errors.New("no workflows provided")

type WorkflowJob struct {
	Name string
}

type Workflow struct {
	Name string
	Jobs []WorkflowJob
}

type TaskFactory struct {
	exec      executor.Executor
	conf      config.Config
	execQueue queue.Queue
}

// NewTaskFactory creates a factory that makes it easier to create tasks for the
// workers
func NewTaskFactory(appContext cli.AppContext, execQueue queue.Queue) (*TaskFactory, error) {
	// Config
	conf, err := config.GetConfig(appContext.Config.ConfigPath)
	if err != nil {
		return nil, err
	}

	tf := TaskFactory{
		exec:      appContext.Executor,
		conf:      *conf,
		execQueue: execQueue,
	}
	return &tf, nil
}

func (f *TaskFactory) CreateTask(ctx context.Context, workflows []Workflow, cid string) ([]*dag.Node[string], error) {
	if len(workflows) == 0 {
		return nil, ErrEmptyWorkflows
	}
	log.Ctx(ctx).Debug().Str("cid", cid).Msg("creating dags")
	var dags []*dag.Node[string]                       // List of dags
	derivativeNode := dag.NewNode(f.buildJob("merge")) // The final merge job, enabled later
	for _, workflow := range workflows {
		log.Ctx(ctx).Debug().Str("workflow", workflow.Name).Msg("adding workflow")
		if len(workflow.Jobs) == 0 {
			return nil, ErrWorkflowNoJobs
		}
		// For each step in the workflow, create a linear dag
		var rootDag *dag.Node[string]
		var childNode *dag.Node[string]
		for _, step := range workflow.Jobs {
			log.Ctx(ctx).Debug().Str("job", step.Name).Msg("creating job")
			j := f.buildJob(step.Name)
			if rootDag == nil { // If this is a new dag, create it
				log.Ctx(ctx).Debug().Str("cid", cid).Msg("new root node")
				rootDag = dag.NewDag(j, []string{cid})
				childNode = rootDag
			} else { // If this is a child, make it the next root node
				log.Ctx(ctx).Debug().Msg("new child")
				c := dag.NewNode(j)
				childNode.AddChild(c)
				childNode = c
			}
		}
		// Add one final merge job to the end of the dag
		if !f.conf.Workflow.DisableDerivative {
			log.Ctx(ctx).Debug().Msg("adding derivative node")
			childNode.AddChild(derivativeNode)
		}
		// Add all dags to a list for later deduplication
		dags = append(dags, rootDag)
		log.Ctx(ctx).Debug().Int("len", len(dags)).Msg("added dag to list")
	}
	return dags, nil
}

func (f *TaskFactory) buildJob(name string) dag.Work[string] {
	return func(ctx context.Context, inputs []string) []string {
		log.Ctx(ctx).Info().Str("step", name).Msg("Running job")
		j := f.render(name, inputs)
		resChan := make(chan executor.Result)
		defer close(resChan)
		err := f.execQueue.Enqueue(func(ctx context.Context) {
			r, err := f.exec.Execute(ctx, j)
			if err != nil {
				log.Fatal().Err(err).Msg("Error executing job")
			}
			resChan <- r
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Error enqueueing job")
		}
		r := <-resChan
		res := inputs
		res = append(res, r.CID.String())
		return res
	}
}

func (f *TaskFactory) render(name string, cids []string) interface{} {
	job, err := f.GetJob(name)
	if err != nil {
		panic(err)
	}

	return f.exec.Render(job, cids)
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
