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

func (f *TaskFactory) CreateTask(ctx context.Context, workflows []Workflow, cid string) ([]*dag.Node[[]string], error) {
	log.Ctx(ctx).Debug().Msgf("Creating task for %s", cid)
	var dags []*dag.Node[[]string]
	for _, workflow := range workflows {
		log.Ctx(ctx).Debug().Msgf("Adding workflow %s", workflow.Name)
		if len(workflow.Jobs) == 0 {
			return nil, ErrWorkflowNoJobs
		}
		var d *dag.Node[[]string]
		for i, step := range workflow.Jobs {
			if i == 0 {
				j := f.buildJob(step.Name)
				d = dag.NewNode(j, []string{cid})
			} else {
				d.AddChild(f.buildJob(step.Name))
			}
		}
		dags = append(dags, d)
	}

	return dags, nil
}

func (f *TaskFactory) buildJob(name string) dag.Work[[]string] {
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
