package queue

import "context"

// Callable encapsulates the method that will be called by a worker to run a job
type Callable func(context.Context) error

// Queuer encapsulates the Enqueue method that will be called by a dispatcher
// Deprecated: use Queue instead
type Queuer interface {
	Enqueue(QueueRepository, string, Callable) error // Adds item to the queue
	Start() error                                    // Starts the processing of the queue
	Stop() error                                     // Stops the processing of the queue
}

// Queuer encapsulates the Enqueue method that will be called by a dispatcher
type Queue interface {
	Enqueue(func(context.Context)) error // Adds item to the queue
	Start()                              // Starts the processing of the queue
	Stop()                               // Stops the processing of the queue
}
