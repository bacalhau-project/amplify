package queue

import (
	"context"
	"testing"
	"time"

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

func TestQueueStopsWhenCtxDeferred(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	q, err := NewGenericQueue(ctx, 1, 1)
	assert.NilError(t, err)
	q.Start()
	defer q.Stop()
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

func TestQueueSizeEnforced(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	q, err := NewGenericQueue(ctx, 1, 1)
	assert.NilError(t, err)
	q.Start()
	defer q.Stop()
	err = q.Enqueue(func(ctx context.Context) { time.Sleep(100 * time.Millisecond) })
	assert.NilError(t, err)
	err = q.Enqueue(func(ctx context.Context) { time.Sleep(100 * time.Millisecond) })
	assert.Equal(t, err, ErrQueueFull)
}
