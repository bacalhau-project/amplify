package queue

import (
	"context"
)

// Queue encapsulates the Enqueue method that will be called by a dispatcher
type Queue interface {
	Enqueue(func(context.Context)) error // Adds item to the queue
	Start()                              // Starts the processing of the queue
	Stop()                               // Stops the processing of the queue
	IsFull() bool
}

func NewMockQueue() Queue {
	return &mockQueue{
		QueueCount: 0,
	}
}

type mockQueue struct {
	QueueCount int
}

func (t *mockQueue) Enqueue(f func(context.Context)) error {
	t.QueueCount += 1
	f(context.Background())
	return nil
}

func (*mockQueue) Start() {
}

func (*mockQueue) Stop() {
}

func (*mockQueue) IsFull() bool {
	return false
}
