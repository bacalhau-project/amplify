// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"
	"errors"
	"fmt"

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
var ErrNoRootNodes = errors.New("no root nodes found, please check your config")

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

// Create tasks forms the DAG of all the work. The number of inputs (len(cid))
// must be equal to the number of outputs (len(dag.Nodes)), otherwise it ends
// up running multiple times.
func (f *TaskFactory) CreateTask(ctx context.Context, workflowFilter string, cid string) (*dag.Node[dag.IOSpec], error) {
	log.Ctx(ctx).Debug().Str("cid", cid).Msg("building dags")

	// TODO: Figure out if we want to filter for a specific workflow

	// Start with the root node that represents the CID
	rootWork := func(context.Context, []dag.IOSpec, chan dag.NodeStatus) []dag.IOSpec {
		return []dag.IOSpec{dag.NewIOSpec("root", "root", cid, "", true)}
	}
	rootNode := dag.NewDag(cid, rootWork, nil)

	// Build up the dag until we've added steps
	dags := make(map[string]*dag.Node[dag.IOSpec], len(f.conf.Nodes))
	for {
		var steps []config.Node
		for _, s := range f.conf.Nodes {
			if _, ok := dags[s.ID]; !ok {
				steps = append(steps, s)
			}
		}
		if len(steps) == 0 {
			break
		}
		for _, step := range steps {
			log.Ctx(ctx).Debug().Int("len(dags)", len(dags)).Msg("built dags")
			inputsReady := true
			for _, i := range step.Inputs {
				if i.Root {
					continue
				}
				if _, ok := dags[i.StepID]; !ok {
					inputsReady = false
					break
				}
			}
			if !inputsReady {
				log.Ctx(ctx).Debug().Str("step", step.ID).Msg("inputs not ready")
				continue
			}
			log.Ctx(ctx).Debug().Str("step", step.ID).Msg("inputs ready")
			work := f.buildJob(step)
			dags[step.ID] = dag.NewNode(step.ID, work)
			for _, i := range step.Inputs {
				if i.Root {
					log.Ctx(ctx).Debug().Str("parent", "root").Str("child", step.ID).Msg("adding child")
					rootNode.AddChild(dags[step.ID])
				} else {
					log.Ctx(ctx).Debug().Str("parent", i.StepID).Str("child", step.ID).Msg("adding child")
					dags[i.StepID].AddChild(dags[step.ID])
				}
			}
		}
	}
	if len(rootNode.Children()) == 0 {
		return nil, ErrNoRootNodes
	}
	return rootNode, nil
}

func (f *TaskFactory) buildJob(step config.Node) dag.Work[dag.IOSpec] {
	return func(ctx context.Context, inputs []dag.IOSpec, statusChan chan dag.NodeStatus) []dag.IOSpec {
		log.Ctx(ctx).Info().Str("jobID", step.JobID).Msg("Running job")
		// The inputs presented here are actually a copy of the previous node's
		// outputs. So we need to re-compute to make sure that the Bacalau job
		// is presented with the right values
		var computedInputs []executor.ExecutorIOSpec
		for _, stepInput := range step.Inputs {
			// If step input expects a root input, find the associated root input
			if stepInput.Root {
				for _, actualInput := range inputs {
					if !actualInput.IsRoot() {
						continue
					}
					computedInputs = append(computedInputs, executor.ExecutorIOSpec{
						Ref:  actualInput.CID(),
						Path: stepInput.Path,
					})
				}
			}

			// If it's not a root input, check the actual inputs for a matching
			// step ID and output ID
			for _, actualInput := range inputs {
				if actualInput.IsRoot() {
					continue
				}
				if actualInput.NodeID() == stepInput.StepID && actualInput.ID() == stepInput.OutputId {
					computedInputs = append(computedInputs, executor.ExecutorIOSpec{
						Name: fmt.Sprintf("%s-%s", actualInput.NodeID(), actualInput.ID()),
						Ref:  actualInput.CID(),
						Path: stepInput.Path,
					})
				}
			}
		}
		if len(computedInputs) != len(step.Inputs) {
			log.Ctx(ctx).Error().Int("len(computedInputs)", len(computedInputs)).Int("len(step.Inputs)", len(step.Inputs)).Str("node_id", step.ID).Msg("problem computing inputs for node, please check the config")
		}
		var computedOutputs []executor.ExecutorIOSpec
		for _, o := range step.Outputs {
			computedOutputs = append(computedOutputs, executor.ExecutorIOSpec{
				Name: o.ID,
				Path: o.Path,
			})
		}
		j := f.render(step.JobID, computedInputs, computedOutputs)
		resChan := make(chan executor.Result)
		defer close(resChan)
		err := f.execQueue.Enqueue(func(ctx context.Context) {
			r, err := f.exec.Execute(ctx, j)
			if err != nil {
				log.Fatal().Err(err).Msg("Error executing job")
			}
			// TODO: in the future make node status' more regular by adding to the Execute method
			statusChan <- dag.NodeStatus{
				ID:     r.ID,
				StdOut: r.StdOut,
				StdErr: r.StdErr,
				Status: r.Status,
			}
			resChan <- r
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Error enqueueing job")
		}
		r := <-resChan

		if len(step.Outputs) > 0 {
			results := make([]dag.IOSpec, 1) // TODO: Only works with zero'th output
			results[0] = dag.NewIOSpec(step.ID, step.Outputs[0].ID, r.CID.String(), step.Outputs[0].Path, false)
			return results
		} else {
			return []dag.IOSpec{}
		}
	}
}

func (f *TaskFactory) render(jobID string, inputs []executor.ExecutorIOSpec, outputs []executor.ExecutorIOSpec) interface{} {
	job, err := f.GetJob(jobID)
	if err != nil {
		panic(err)
	}
	return f.exec.Render(job, inputs, outputs)
}

// GetJob gets a job config from a job factory
func (f *TaskFactory) GetJob(name string) (config.Job, error) {
	for _, job := range f.conf.Jobs {
		if job.ID == name {
			return job, nil
		}
	}
	return config.Job{}, ErrJobNotFound
}

// JobNames returns all the names of the jobs in a job factory
func (f *TaskFactory) JobNames() []string {
	var names []string
	for _, job := range f.conf.Jobs {
		names = append(names, job.ID)
	}
	return names
}

func (f *TaskFactory) GetNode(step string) (config.Node, error) {
	for _, w := range f.conf.Nodes {
		if w.ID == step {
			return w, nil
		}
	}
	return config.Node{}, ErrWorkflowNotFound
}

func (f *TaskFactory) NodeNames() []string {
	var workflows []string
	for _, w := range f.conf.Nodes {
		workflows = append(workflows, w.ID)
	}
	return workflows
}
