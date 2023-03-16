package task

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"gotest.tools/assert"
)

func TestNewTaskFactory(t *testing.T) {
	tf, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: "../../config.yaml"}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, tf != nil)

	_, err = NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: "nonexistent.yaml"}}, &mockQueue{})
	assert.Assert(t, err != nil)
}

func TestTaskFactory_CreateTask(t *testing.T) {
	q := &mockQueue{}
	tf, err := NewTaskFactory(cli.AppContext{Executor: &mockExecutor{}, Config: &config.AppConfig{ConfigPath: "../../config.yaml"}}, q)
	assert.NilError(t, err)

	// Simple workflow
	d, err := tf.CreateTask(context.Background(), Workflow{Name: "test", Jobs: []WorkflowJob{{Name: "metadata"}}}, "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	assert.Equal(t, len(d.Children()), 0)
	assert.Assert(t, !d.Meta().CreatedAt.IsZero())
	d.Execute(context.Background())
	assert.Equal(t, q.counter, 1)

	// Workflow with children
	d, err = tf.CreateTask(context.Background(), Workflow{Name: "test", Jobs: []WorkflowJob{{Name: "metadata"}, {Name: "metadata"}}}, "cid")
	assert.NilError(t, err)
	assert.Equal(t, len(d.Children()), 1)
	d.Execute(context.Background())
	assert.Equal(t, q.counter, 3)
}

func TestTaskFactory_GetJob(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- name: metadata 
  image: some-image
  entrypoint: ["extract-metadata", "/inputs", "/outputs"] # Container entrypoint
`), 0644)
	assert.NilError(t, err)
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	j, err := valid.GetJob("metadata")
	assert.NilError(t, err)
	assert.Assert(t, len(j.Entrypoint) > 0)
	_, err = valid.GetJob("nogood")
	assert.ErrorContains(t, err, "not found")
}

func TestTaskFactory_JobNames(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- name: test
- name: test2
`), 0644)
	assert.NilError(t, err)
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	assert.Assert(t, reflect.DeepEqual(valid.JobNames(), []string{"test", "test2"}))
}

func TestTaskFactory_GetWorkflow(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`workflows:
- name: test
  jobs:
  - name: metadata
- name: test2
`), 0644)
	assert.NilError(t, err)
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	w, err := valid.GetWorkflow("test")
	assert.NilError(t, err)
	assert.Assert(t, len(w.Jobs) > 0)
	_, err = valid.GetWorkflow("nogood")
	assert.ErrorContains(t, err, "not found")
}

func TestTaskFactory_WorkflowNames(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`workflows:
- name: test
- name: test2
`), 0644)
	assert.NilError(t, err)
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	assert.Assert(t, reflect.DeepEqual(valid.WorkflowNames(), []string{"test", "test2"}))
}

var _ queue.Queue = (*mockQueue)(nil)

type mockQueue struct {
	counter int
}

func (q *mockQueue) Enqueue(w func(context.Context)) error {
	q.counter++
	go func() {
		w(context.Background())
	}()
	return nil
}

func (*mockQueue) Start() {
}

func (*mockQueue) Stop() {
}

var _ executor.Executor = (*mockExecutor)(nil)

type mockExecutor struct{}

func (*mockExecutor) Execute(context.Context, interface{}) (executor.Result, error) {
	return executor.Result{}, nil
}

func (*mockExecutor) Render(config.Job, []string) interface{} {
	return ""
}
