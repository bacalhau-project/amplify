package workflow

import (
	"errors"

	"github.com/bacalhau-project/amplify/pkg/config"
)

var ErrWorkflowNotFound = errors.New("workflow not found")

type WorkflowFactory struct {
	conf config.Config
}

type WorkflowJob struct {
	Name string
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
	return Workflow{}, ErrWorkflowNotFound
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
		w.Jobs = append(w.Jobs, WorkflowJob{
			Name: j.Name,
		})
	}
	return w, nil
}
