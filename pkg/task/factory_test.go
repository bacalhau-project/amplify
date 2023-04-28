package task

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/executor"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestNewTaskFactory(t *testing.T) {
	tf, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: "../../config.yaml"}}, &mockQueue{}, NewMockNodeStore())
	assert.NilError(t, err)
	assert.Assert(t, tf != nil)
	_, err = NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: "nonexistent.yaml"}}, &mockQueue{}, NewMockNodeStore())
	assert.Assert(t, err != nil)
}

func TestTaskFactory_CreateTask(t *testing.T) {
	ctx := context.Background()
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
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	// Simple workflow
	d, err := tf.CreateTask(ctx, uuid.New(), "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(root.Children), 1) // One child because of final derivative job
	assert.Assert(t, !root.Metadata.CreatedAt.IsZero())
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(2))
}

func waitForAndAssert(t *testing.T, f func() int32, expected int32) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		if f() == 2 || ctx.Err() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, f(), expected)
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
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)
	_, err = tf.CreateTask(context.Background(), uuid.New(), "cid")
	assert.ErrorContains(t, err, ErrNoRootNodes.Error())
}

func TestTaskFactory_MergeTask(t *testing.T) {
	ctx := context.Background()
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
  - node_id: first
    output_id: default
    path: /inputs/first
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
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	d, err := tf.CreateTask(ctx, uuid.New(), "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(root.Children), 2)
	assert.Assert(t, !root.Metadata.CreatedAt.IsZero())
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(3))
	child, err := root.Children[1].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(child.Inputs), 2)
}

func TestTaskFactory_DisconnectedNodes(t *testing.T) {
	ctx := context.Background()
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
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)
	_, err = tf.CreateTask(ctx, uuid.New(), "cid")
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
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{}, NewMockNodeStore())
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
	valid, err := NewTaskFactory(cli.AppContext{Config: &config.AppConfig{ConfigPath: tempFile}}, &mockQueue{}, NewMockNodeStore())
	assert.NilError(t, err)
	assert.Assert(t, valid != nil)
	assert.Assert(t, reflect.DeepEqual(valid.JobNames(), []string{"test", "test2"}))
}

func TestTaskFactory_CreateTaskWithBlockingPredicate(t *testing.T) {
	ctx := context.Background()
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: job
graph:
- id: first
  job_id: job
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: second
  job_id: job
  inputs:
  - node_id: first
    output_id: default
    path: /inputs/first
    predicate: (image\/|video\/).+
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: third
  job_id: job
  inputs:
  - node_id: second
    output_id: default
    path: /inputs/second
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	// Simple workflow
	d, err := tf.CreateTask(ctx, uuid.New(), "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(1))
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	child, err := root.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child.Results.Skipped, true)
	child2, err := child.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child2.Results.Skipped, true)
}

func TestTaskFactory_CreateTaskWithMatchingPredicate(t *testing.T) {
	ctx := context.Background()
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: job
graph:
- id: first
  job_id: job
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: second
  job_id: job
  inputs:
  - node_id: first
    output_id: default
    path: /inputs/first
    predicate: (image\/|video\/).+
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{
		StdOut: "image/png",
	}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	d, err := tf.CreateTask(ctx, uuid.New(), "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(2))
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	child, err := root.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child.Results.Skipped, false)
}

func TestTaskFactory_CreateTaskWithForkingPredicate(t *testing.T) {
	ctx := context.Background()
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: job
graph:
- id: first
  job_id: job
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: second
  job_id: job
  inputs:
  - node_id: first
    output_id: default
    path: /inputs/first
    predicate: (image\/|video\/).+
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: third
  job_id: job
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
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"": &mockExecutor{}}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	d, err := tf.CreateTask(ctx, uuid.New(), "cid")
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(2))
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(root.Children), 2)
	child1, err := root.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(child1.Children), 1)
	child2, err := root.Children[1].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child1.Results.Skipped, true)
	assert.Equal(t, child2.Results.Skipped, false)
	child3, err := child1.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(child3.Inputs), 1)
}

func TestTaskFactory_CreateTaskWithRootInternalNode(t *testing.T) {
	ctx := context.Background()
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: job
  type: bacalhau
- id: root
  type: internal
  internal_job_id: root-job
graph:
- id: first
  job_id: root
  inputs:
  - root: true # Identifies that this is a root node
    path: /inputs # Path where inputs will be placed
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
- id: second
  job_id: job
  inputs:
  - node_id: first
    output_id: default
    path: /inputs/first
  outputs:
  - id: default # Id of output
    path: /outputs # Path of output
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"bacalhau": &mockExecutor{}, "internal": executor.NewInternalExecutor()}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	// Simple workflow
	cid := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	d, err := tf.CreateTask(ctx, uuid.New(), cid)
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(2))
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, root.Outputs[0].CID(), cid)
	child, err := root.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child.Results.Skipped, false)
}

func TestTaskFactory_CreateTaskWithDefaults(t *testing.T) {
	ctx := context.Background()
	q := &mockQueue{}
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, []byte(`jobs:
- id: job
  type: bacalhau
- id: root
  type: internal
  internal_job_id: root-job
graph:
- id: first
  job_id: root
  inputs:
  - root: true
    path: /inputs
  outputs:
  - path: /outputs
- id: second
  job_id: job
  inputs:
  - node_id: first
    path: /inputs/first
  outputs:
  - path: /outputs # Path of output
- id: third
  job_id: job
  inputs:
  - node_id: second
    predicate: ".*"
  outputs:
  - path: /outputs # Path of output
- id: fourth
  job_id: job
  inputs:
  - node_id: third
    predicate: "hello"
  outputs:
  - path: /outputs # Path of output
- id: fifth
  job_id: job
  inputs:
  - node_id: fourth
`), 0644)
	assert.NilError(t, err)
	tf, err := NewTaskFactory(cli.AppContext{Executor: map[string]executor.Executor{"bacalhau": &mockExecutor{}, "internal": executor.NewInternalExecutor()}, Config: &config.AppConfig{ConfigPath: tempFile}}, q, NewMockNodeStore())
	assert.NilError(t, err)

	// Simple workflow
	cid := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	d, err := tf.CreateTask(ctx, uuid.New(), cid)
	assert.NilError(t, err)
	assert.Assert(t, d != nil)
	e, _ := dag.NewNodeExecutor[dag.IOSpec](ctx, nil)
	e.Execute(ctx, uuid.New(), d[0])
	waitForAndAssert(t, q.Count, int32(3)) // Three nodes, should not skip
	root, err := d[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, root.Outputs[0].CID(), cid)
	assert.Equal(t, root.Outputs[0].ID(), "default")
	child1, err := root.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child1.Inputs[0].ID(), "default")
	assert.Equal(t, child1.Results.StdErr, "1 input and 1 output")
	child2, err := child1.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, child2.Inputs[0].ID(), "default")
	assert.Equal(t, len(child2.Outputs), 1)
	assert.Equal(t, child2.Results.Skipped, false)
	assert.Equal(t, child2.Results.StdErr, "0 input and 1 output")
}

var _ queue.Queue = (*mockQueue)(nil)

type mockQueue struct {
	counter atomic.Int32
}

func (q *mockQueue) Count() int32 {
	return q.counter.Load()
}
func (q *mockQueue) Enqueue(w func(context.Context)) error {
	q.counter.Add(1)
	go func() {
		w(context.Background())
	}()
	return nil
}

func (*mockQueue) Start() {
}

func (*mockQueue) Stop() {
}

func (*mockQueue) IsFull() bool {
	return false
}

var _ executor.Executor = (*mockExecutor)(nil)

type mockExecutor struct {
	StdOut  string
	inputs  []executor.ExecutorIOSpec
	outputs []executor.ExecutorIOSpec
}

func (m *mockExecutor) Execute(context.Context, config.Job, interface{}) (executor.Result, error) {
	return executor.Result{
		StdOut: m.StdOut,
		StdErr: fmt.Sprintf("%d input and %d output", len(m.inputs), len(m.outputs)),
	}, nil
}

func (m *mockExecutor) Render(j config.Job, inputs []executor.ExecutorIOSpec, outputs []executor.ExecutorIOSpec) (interface{}, error) {
	m.inputs = inputs
	m.outputs = outputs
	return "", nil
}

func NewMockNodeStore() dag.NodeStore[dag.IOSpec] {
	persistence := db.NewInMemDB()
	wr := dag.NewInMemWorkRepository[dag.IOSpec]()
	ns, _ := dag.NewNodeStore(
		context.Background(),
		persistence,
		wr,
	)
	return ns
}
