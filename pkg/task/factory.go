// A task encapsulates the logic of iterating or using composites for either
// a job or a workflow.
package task

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/rs/zerolog/log"
)

var ErrJobNotFound = errors.New("job not found")
var ErrWorkflowNotFound = errors.New("workflow not found")
var ErrWorkflowNoJobs = errors.New("workflow has no jobs")
var ErrEmptyWorkflows = errors.New("no workflows provided")
var ErrNoRootNodes = errors.New("no root nodes found, please check your config")
var ErrDisconnectedNode = errors.New("node expected by input doesn't exist")
var ErrExecutorNotFound = errors.New("executor type not found")

// TODO: This is a limitation. No reason why we can't have multiple root nodes.
var ErrTooManyRootNodes = errors.New("too many root nodes found, amplify only works with one root node")

type TaskFactory struct {
	exec      map[string]executor.Executor // Map of executor types to implementations
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
	dags := make(map[string]*dag.Node[dag.IOSpec], len(f.conf.Graph))
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
			dags[step.ID] = dag.NewNode(step.ID, work)
			for _, i := range step.Inputs {
				if i.Root {
					log.Ctx(ctx).Debug().Str("parent", "root").Str("child", step.ID).Msg("adding child")
					dags[step.ID].AddRootInput(
						dag.NewIOSpec("root", "root", cid, "", true, ""),
					)
				} else {
					log.Ctx(ctx).Debug().Str("parent", i.NodeID).Str("child", step.ID).Msg("adding child")
					dags[i.NodeID].AddChild(dags[step.ID])
				}
			}
		}
	}
	var rootNode []*dag.Node[dag.IOSpec]
	for _, node := range dags {
		if node.IsRoot() {
			rootNode = append(rootNode, node)
		}
	}
	if len(rootNode) == 0 {
		return nil, ErrNoRootNodes
	}
	if len(rootNode) > 1 {
		return nil, ErrTooManyRootNodes
	}
	return rootNode[0], nil
}

func (f *TaskFactory) buildJob(step config.Node) dag.Work[dag.IOSpec] {
	return func(ctx context.Context, inputs []dag.IOSpec, statusChan chan dag.NodeStatus) []dag.IOSpec {
		defer close(statusChan) // Must close the channel to signify the end of status updates
		log.Ctx(ctx).Info().Str("jobID", step.JobID).Msg("Starting job")
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
				if actualInput.NodeID() == stepInput.NodeID && actualInput.ID() == stepInput.OutputID {
					if stepInput.Predicate != "" {
						b, err := regexp.MatchString(stepInput.Predicate, actualInput.Context())
						if err != nil {
							log.Ctx(ctx).Error().Err(err).Msg("error regexing predicate")
						} else if !b {
							log.Ctx(ctx).Info().Str("predicate", stepInput.Predicate).Str("context", actualInput.Context()).Msg("predicate didn't match, skipping")
							statusChan <- dag.NodeStatus{
								Status:  "Skipped",
								StdErr:  fmt.Sprintf("skipped due to predicate: %s", stepInput.Predicate),
								Skipped: true,
							}
							return []dag.IOSpec{}
						}
					}
					i := executor.ExecutorIOSpec{
						Name: fmt.Sprintf("%s-%s", actualInput.NodeID(), actualInput.ID()),
						Ref:  actualInput.CID(),
						Path: stepInput.Path,
					}
					log.Ctx(ctx).Debug().Str("input", i.Name).Str("ref", i.Ref).Str("path", i.Path).Msg("input")
					computedInputs = append(computedInputs, i)
				}
			}
		}

		if len(computedInputs) == 0 {
			log.Ctx(ctx).Info().Str("step", step.ID).Msg("no inputs found, skipping")
			statusChan <- dag.NodeStatus{
				StdErr:  "skipped due to no inputs",
				Status:  "Skipped",
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
				log.Warn().Err(err).Msg("Error executing job")
			}
			// TODO: in the future make node status' more regular by adding to the Execute method
			statusChan <- dag.NodeStatus{
				ID:     r.ID,
				StdOut: r.StdOut,
				StdErr: r.StdErr,
				Status: r.Status,
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
			results[0] = dag.NewIOSpec(step.ID, step.Outputs[0].ID, r.CID.String(), step.Outputs[0].Path, false, r.StdOut)
			return results
		} else {
			return []dag.IOSpec{}
		}
	}
}

func (f *TaskFactory) execute(ctx context.Context, jobID string, inputs []executor.ExecutorIOSpec, outputs []executor.ExecutorIOSpec) (executor.Result, error) {
	job, err := f.GetJob(jobID)
	if err != nil {
		panic(err)
	}
	if _, ok := f.exec[job.Type]; !ok {
		return executor.Result{
			StdErr: ErrExecutorNotFound.Error(),
			Status: "error",
		}, ErrExecutorNotFound
	}
	// TODO: probably don't need the render step any more. Simplify to just execute.
	j := f.exec[job.Type].Render(job, inputs, outputs)
	return f.exec[job.Type].Execute(ctx, j)
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
	for _, w := range f.conf.Graph {
		if w.ID == step {
			return w, nil
		}
	}
	return config.Node{}, ErrWorkflowNotFound
}

func (f *TaskFactory) NodeNames() []string {
	var workflows []string
	for _, w := range f.conf.Graph {
		workflows = append(workflows, w.ID)
	}
	return workflows
}
