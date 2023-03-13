package queue

import "context"

// Queue encapsulates the Enqueue method that will be called by a dispatcher
type Queue interface {
	Enqueue(func(context.Context)) error // Adds item to the queue
	Start()                              // Starts the processing of the queue
	Stop()                               // Stops the processing of the queue
}
