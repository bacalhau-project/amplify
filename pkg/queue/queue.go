package queue

import "context"

// Callable encapsulates the method that will be called by a worker to run a job
type Callable func(context.Context) error

// Queuer encapsulates the Enqueue method that will be called by a dispatcher
type Queuer interface {
	Enqueue(QueueRepository, string, Callable) error // Adds item to the queue
	Start() error                                    // Starts the processing of the queue
	Stop() error                                     // Stops the processing of the queue
}
