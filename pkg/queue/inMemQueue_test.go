package queue

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"gotest.tools/assert"
)

func TestQueueLifecycle(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	q, err := NewGenericQueue(ctx, 1, 1)
	assert.NilError(t, err)
	q.Start()
	defer q.Stop()
	cancelFunc()
}

func TestQueueWorker(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()
	q, err := NewGenericQueue(ctx, 1, 1)
	assert.NilError(t, err)
	q.Start()
	defer q.Stop()
	called := make(chan bool)
	job := func(ctx context.Context) {
		called <- true
	}
	err = q.Enqueue(job)
	assert.NilError(t, err)
	for {
		select {
		case <-called:
			return
		case <-ctx.Done():
			assert.Assert(t, false)
		}
	}
}

func TestQueueWithDag(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()
	q, err := NewGenericQueue(ctx, 1, 1)
	assert.NilError(t, err)
	q.Start()
	defer q.Stop()
	root := dag.NewDag(func(ctx context.Context, inputs []int) []int {
		return []int{1}
	}, []int{0})
	child1 := dag.NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] + 1}
	})
	root.AddChild(child1)
	child2 := dag.NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] + 1}
	})
	child1.AddChild(child2)
	root.Execute(ctx)
	err = q.Enqueue(root.Execute)
	assert.NilError(t, err)
	for {
		if root.Outputs()[0] != 0 || ctx.Err() != nil {
			break
		}
	}
	assert.Equal(t, root.Outputs()[0], 1)
	assert.Equal(t, root.Children()[0].Outputs()[0], 2)
	assert.Equal(t, root.Children()[0].Children()[0].Outputs()[0], 3)
}
