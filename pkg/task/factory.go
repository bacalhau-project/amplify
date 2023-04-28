// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var ErrJobNotFound = errors.New("job not found")
var ErrWorkflowNotFound = errors.New("workflow not found")
var ErrWorkflowNoJobs = errors.New("workflow has no jobs")
var ErrEmptyWorkflows = errors.New("no workflows provided")
var ErrNoRootNodes = errors.New("no root nodes found, please check your config")
var ErrDisconnectedNode = errors.New("node expected by input doesn't exist")
var ErrExecutorNotFound = errors.New("executor type not found")
var ErrRenderingJob = errors.New("error rendering job")

type TaskFactory interface {
	CreateTask(ctx context.Context, executionID uuid.UUID, cid string) ([]dag.Node[dag.IOSpec], error)
	JobNames() []string
	NodeNames() []string
	GetNode(step string) (config.Node, error)
	GetJob(name string) (config.Job, error)
}

type taskFactory struct {
	exec        map[string]executor.Executor // Map of executor types to implementations
	conf        config.Config
	execQueue   queue.Queue
	nodeFactory dag.NodeStore[dag.IOSpec]
}

// NewTaskFactory creates a factory that makes it easier to create tasks for the
// workers
func NewTaskFactory(appContext cli.AppContext, execQueue queue.Queue, nodeFactory dag.NodeStore[dag.IOSpec]) (TaskFactory, error) {
	// Config
	conf, err := config.GetConfig(appContext.Config.ConfigPath)
	if err != nil {
		return nil, err
	}

	return &taskFactory{
		exec:        appContext.Executor,
		conf:        *conf,
		execQueue:   execQueue,
		nodeFactory: nodeFactory,
	}, nil
}

// Create tasks forms the DAG of all the work. The number of inputs (len(cid))
// must be equal to the number of outputs (len(dag.Nodes)), otherwise it ends
// up running multiple times.
func (f *taskFactory) CreateTask(ctx context.Context, executionID uuid.UUID, cid string) ([]dag.Node[dag.IOSpec], error) {
	log.Ctx(ctx).Debug().Str("cid", cid).Msg("building dags")

	// Check that all step inputs actually exist, otherwise we'll loop infinitely
	nodeNames := make([]string, len(f.conf.Graph))
	for i, node := range f.conf.Graph {
		nodeNames[i] = node.ID
	}
	for _, node := range f.conf.Graph {
		for _, input := range node.Inputs {
			if input.Root {
				continue
			}
			if !util.Contains(nodeNames, input.NodeID) {
				return nil, fmt.Errorf("%s: input %s for step %s does not exist", ErrDisconnectedNode, input.NodeID, node.ID)
			}
		}
	}

	// Build up the dag until we've added steps
	dags := make(map[string]dag.Node[dag.IOSpec], len(f.conf.Graph))
	for {
		var steps []config.Node
		for _, s := range f.conf.Graph {
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
				if _, ok := dags[i.NodeID]; !ok {
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
			newNode, err := f.nodeFactory.NewNode(ctx, dag.NodeSpec[dag.IOSpec]{
				Name:    step.ID,
				OwnerID: executionID,
				Work:    work,
			})
			if err != nil {
				return nil, err
			}
			dags[step.ID] = newNode
			for _, i := range step.Inputs {
				if i.Root {
					log.Ctx(ctx).Debug().Str("parent", "root").Str("child", step.ID).Msg("adding child")
					err = dags[step.ID].AddInput(ctx, dag.NewIOSpec(i.NodeID, "root", cid, "", true, ""))
					if err != nil {
						return nil, err
					}
				} else {
					log.Ctx(ctx).Debug().Str("parent", i.NodeID).Str("child", step.ID).Msg("adding child")
					err = dags[i.NodeID].AddParentChildRelationship(ctx, dags[step.ID])
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	rootNodes, err := dag.FilterForRootNodes(ctx, dag.NodeMapToList(dags))
	if err != nil {
		return nil, err
	}
	if len(rootNodes) == 0 {
		return nil, ErrNoRootNodes
	}
	return rootNodes, nil
}

func (f *taskFactory) buildJob(step config.Node) dag.Work[dag.IOSpec] {
	return func(ctx context.Context, inputs []dag.IOSpec, resultChan chan dag.NodeResult) []dag.IOSpec {
		defer close(resultChan) // Must close the channel to signify the end of status updates

		// Ensure context hasn't been cancelled
		if ctx.Err() != nil {
			log.Ctx(ctx).Error().Err(ctx.Err()).Msg("Context cancelled")
			return nil
		}

		step.ApplyDefaults()

		// The inputs presented here are actually a copy of the previous node's
		// outputs. So we need to re-compute to make sure that the Bacalau job
		// is presented with the right values
		var computedInputs []executor.ExecutorIOSpec
		inputsWithPredicates := 0
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
				if actualInput.NodeName() == stepInput.NodeID && actualInput.ID() == stepInput.OutputID {
					if stepInput.Predicate != "" {
						b, err := regexp.MatchString(stepInput.Predicate, actualInput.Context())
						if err != nil {
							log.Ctx(ctx).Error().Err(err).Msg("error regexing predicate")
						} else if !b {
							log.Ctx(ctx).Info().Str("predicate", stepInput.Predicate).Str("context", actualInput.Context()).Msg("predicate didn't match, skipping")
							resultChan <- dag.NodeResult{
								StdErr:  fmt.Sprintf("skipped due to predicate: %s", stepInput.Predicate),
								Skipped: true,
							}
							return []dag.IOSpec{}
						}
						inputsWithPredicates++
					}
					// Only create input if it has a path specified
					if stepInput.Path != "" {
						i := executor.ExecutorIOSpec{
							Name: fmt.Sprintf("%s-%s", actualInput.NodeName(), actualInput.ID()),
							Ref:  actualInput.CID(),
							Path: stepInput.Path,
						}
						log.Ctx(ctx).Debug().Str("input", i.Name).Str("ref", i.Ref).Str("path", i.Path).Msg("input")
						computedInputs = append(computedInputs, i)
					}
				}
			}
		}

		if len(computedInputs) == 0 && inputsWithPredicates == 0 {
			log.Ctx(ctx).Info().Str("step", step.ID).Msg("no inputs found, skipping")
			resultChan <- dag.NodeResult{
				StdErr:  "skipped due to no inputs",
				Skipped: true,
			}
			return []dag.IOSpec{}
		}
		var computedOutputs []executor.ExecutorIOSpec
		for _, o := range step.Outputs {
			computedOutputs = append(computedOutputs, executor.ExecutorIOSpec{
				Name: o.ID,
				Path: o.Path,
			})
		}

		resChan := make(chan executor.Result)
		defer close(resChan)
		err := f.execQueue.Enqueue(func(ctx context.Context) {
			log.Ctx(ctx).Info().Str("jobID", step.JobID).Msg("Executing job")
			r, err := f.execute(ctx, step.JobID, computedInputs, computedOutputs)
			if err != nil {
				log.Warn().Err(err).Str("external_id", r.ID).Str("job_id", step.JobID).Str("node_id", step.ID).Msg("error executing job")
			}
			// TODO: in the future make node status' more regular by adding to the Execute method
			resultChan <- dag.NodeResult{
				ID:     r.ID,
				StdOut: r.StdOut,
				StdErr: r.StdErr,
			}
			resChan <- r
			log.Ctx(ctx).Info().Str("jobID", step.JobID).Msg("Finished job")
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Error enqueueing job")
		}
		r := <-resChan

		if len(step.Outputs) > 0 {
			results := make([]dag.IOSpec, 1) // TODO: Only works with zero'th output
			results[0] = dag.NewIOSpec(step.ID, step.Outputs[0].ID, r.CID, step.Outputs[0].Path, false, r.StdOut)
			return results
		} else {
			return []dag.IOSpec{}
		}
	}
}

func (f *taskFactory) execute(ctx context.Context, jobID string, inputs []executor.ExecutorIOSpec, outputs []executor.ExecutorIOSpec) (executor.Result, error) {
	job, err := f.GetJob(jobID)
	if err != nil {
		panic(err)
	}
	if _, ok := f.exec[job.Type]; !ok {
		return executor.Result{
			StdErr: ErrExecutorNotFound.Error(),
			Status: model.JobStateError.String(),
		}, ErrExecutorNotFound
	}
	// TODO: probably don't need the render step any more. Simplify to just execute.
	j, err := f.exec[job.Type].Render(job, inputs, outputs)
	if err != nil {
		return executor.Result{
			StdErr: fmt.Sprintf("%s: %s", ErrRenderingJob.Error(), err.Error()),
			Status: model.JobStateError.String(),
		}, err
	}
	return f.exec[job.Type].Execute(ctx, job, j)
}

// GetJob gets a job config from a job factory
func (f *taskFactory) GetJob(name string) (config.Job, error) {
	for _, job := range f.conf.Jobs {
		if job.ID == name {
			if job.Timeout == 0 {
				job.Timeout = 10 * time.Minute
			}
			return job, nil
		}
	}
	return config.Job{}, ErrJobNotFound
}

// JobNames returns all the names of the jobs in a job factory
func (f *taskFactory) JobNames() []string {
	var names []string
	for _, job := range f.conf.Jobs {
		names = append(names, job.ID)
	}
	return names
}

func (f *taskFactory) GetNode(step string) (config.Node, error) {
	for _, w := range f.conf.Graph {
		if w.ID == step {
			return w, nil
		}
	}
	return config.Node{}, ErrWorkflowNotFound
}

func (f *taskFactory) NodeNames() []string {
	var workflows []string
	for _, w := range f.conf.Graph {
		workflows = append(workflows, w.ID)
	}
	return workflows
}

func NewMockTaskFactory(persistence db.Persistence) TaskFactory {
	return &mockTaskFactory{
		persistence: persistence,
	}
}

type mockTaskFactory struct {
	persistence db.Persistence
}

func (*mockTaskFactory) GetJob(name string) (config.Job, error) {
	panic("unimplemented")
}

func (*mockTaskFactory) GetNode(step string) (config.Node, error) {
	panic("unimplemented")
}

func (*mockTaskFactory) JobNames() []string {
	panic("unimplemented")
}

func (*mockTaskFactory) NodeNames() []string {
	panic("unimplemented")
}

func (f *mockTaskFactory) CreateTask(ctx context.Context, executionID uuid.UUID, cid string) ([]dag.Node[dag.IOSpec], error) {
	wr := dag.NewInMemWorkRepository[dag.IOSpec]()
	root, err := dag.NewNode(ctx, f.persistence, wr, dag.NodeSpec[dag.IOSpec]{
		OwnerID: executionID,
		Name:    "root",
		Work:    dag.NilFunc,
	})
	if err != nil {
		return nil, err
	}
	err = root.AddInput(
		ctx,
		dag.NewIOSpec("root", "input", cid, "/input", true, ""),
	)
	if err != nil {
		return nil, err
	}
	child, err := dag.NewNode(ctx, f.persistence, wr, dag.NodeSpec[dag.IOSpec]{
		OwnerID: executionID,
		Name:    "child",
		Work:    dag.NilFunc,
	})
	if err != nil {
		return nil, err
	}
	err = root.AddInput(
		ctx,
		dag.NewIOSpec("not-root", "input", cid, "/input", false, ""),
	)
	if err != nil {
		return nil, err
	}
	err = root.AddChild(ctx, child)
	if err != nil {
		return nil, err
	}
	return []dag.Node[dag.IOSpec]{root}, nil
}
