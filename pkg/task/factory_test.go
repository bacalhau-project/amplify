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
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: metadata-job
  image: some-image
  entrypoint: ["extract-metadata", "/inputs", "/outputs"] # Container entrypoint
graph:
- id: metadata-node # ID of the step
  job_id: metadata-job # ID of the job it runs
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: metadata-node-2 # ID of the step
  job_id: metadata-job # ID of the job it runs
  inputs:
  - node_id: metadata-node
    output_id: default
    path: /inputs/metadata
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: &mockExecutor{}, Config: &config.AppConfig{ConfigPath: tempFile}}, q)
	assert.NilError(t, err)

	// Simple workflow
	d, err := tf.CreateTask(context.Background(), "", "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	assert.Equal(t, len(d.Children()), 1) // One child because of final derivative job
	assert.Assert(t, !d.Meta().CreatedAt.IsZero())
	d.Execute(context.Background())
	assert.Equal(t, q.counter, 2)
}

func TestTaskFactory_NoRootTasks(t *testing.T) {
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: test
graph:
- id: first
  job_id: test
  outputs:
  - id: default # Id of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: &mockExecutor{}, Config: &config.AppConfig{ConfigPath: tempFile}}, q)
	assert.NilError(t, err)
	_, err = tf.CreateTask(context.Background(), "", "cid")
	assert.ErrorContains(t, err, ErrNoRootNodes.Error())
}

func TestTaskFactory_MergeTask(t *testing.T) {
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: test
graph:
- id: first
  job_id: test
  inputs:
  - root: true
  outputs:
  - id: default # Id of output
    path: /outputs
- id: second
  job_id: test
  inputs:
  - root: true
  outputs:
  - id: default # Id of output
    path: /outputs
- id: merge
  job_id: test
  inputs:
  - node_id: first
    output_id: default
    path: /inputs/first
  - node_id: second
    output_id: default
    path: /inputs/second
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: &mockExecutor{}, Config: &config.AppConfig{ConfigPath: tempFile}}, q)
	assert.NilError(t, err)

	// Simple workflow
	d, err := tf.CreateTask(context.Background(), "", "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	assert.Equal(t, len(d.Children()), 2)
	assert.Assert(t, !d.Meta().CreatedAt.IsZero())
	d.Execute(context.Background())
	assert.Equal(t, len(d.Children()[0].Children()[0].Inputs()), 2)
}

func TestTaskFactory_DisconnectedNodes(t *testing.T) {
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: metadata-job
  image: some-image
  entrypoint: ["extract-metadata", "/inputs", "/outputs"] # Container entrypoint
graph:
- id: metadata-node # ID of the step
  job_id: metadata-job # ID of the job it runs
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: metadata-node-2 # ID of the step
  job_id: metadata-job # ID of the job it runs
  inputs:
  - node_id: non-existent-node
    output_id: default
    path: /inputs/metadata
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: &mockExecutor{}, Config: &config.AppConfig{ConfigPath: tempFile}}, q)
	assert.NilError(t, err)
	_, err = tf.CreateTask(context.Background(), "", "cid")
	assert.ErrorContains(t, err, ErrDisconnectedNode.Error())
}

func TestTaskFactory_GetJob(t *testing.T) {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: metadata 
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
- id: test
- id: test2
`), 0644)
	assert.NilError(t, err)
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{})
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	assert.Assert(t, reflect.DeepEqual(valid.JobNames(), []string{"test", "test2"}))
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

func (*mockExecutor) Render(config.Job, []executor.ExecutorIOSpec, []executor.ExecutorIOSpec) interface{} {
	return ""
}
