package workflow

import (
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
)

type WorkflowFactory struct {
	conf config.Config
}

type WorkflowJob struct {
	Name string
	Job  job.Runner
}

type Workflow struct {
	Name string
	Jobs []WorkflowJob
}

// NewWorkflowFactory creates a new WorkflowFactory
func NewWorkflowFactory(conf config.Config) WorkflowFactory {
	return WorkflowFactory{
		conf: conf,
	}
}

func (f *WorkflowFactory) GetWorkflow(workflow string) (Workflow, error) {
	for _, w := range f.conf.Workflows {
		if w.Name == workflow {
			return f.createWorkflow(w)
		}
	}
	return Workflow{}, fmt.Errorf("workflow %s not found", workflow)
}

func (f *WorkflowFactory) WorkflowNames() []string {
	var workflows []string
	for _, w := range f.conf.Workflows {
		workflows = append(workflows, w.Name)
	}
	return workflows
}

func (f *WorkflowFactory) createWorkflow(workflow config.Workflow) (Workflow, error) {
	w := Workflow{
		Name: workflow.Name,
	}
	for _, j := range workflow.Jobs {
		var runner job.Runner
		switch j.Type {
		case "single":
			runner = job.SingleJob{}
		case "map":
			runner = job.MapJob{}
		default:
			return Workflow{}, fmt.Errorf("unknown job type %s", j.Type)
		}
		w.Jobs = append(w.Jobs, WorkflowJob{
			Name: j.Name,
			Job:  runner,
		})
	}
	return w, nil
}
