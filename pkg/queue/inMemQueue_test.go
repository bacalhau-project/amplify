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
	q.Enqueue(job)
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
	root := dag.NewNode(func(ctx context.Context, input int) int {
		return 1
	}, 0)
	root.AddChild(func(ctx context.Context, input int) int {
		return input + 1
	}).AddChild(func(ctx context.Context, input int) int {
		return input + 1
	})
	q.Enqueue(root.Execute)
	for {
		if root.Output() != 0 || ctx.Err() != nil {
			break
		}
	}
	assert.Equal(t, root.Output(), 1)
	assert.Equal(t, root.Children()[0].Output(), 2)
	assert.Equal(t, root.Children()[0].Children()[0].Output(), 3)
}
