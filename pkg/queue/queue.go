package queue

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/dag"
)

// Queue encapsulates the Enqueue method that will be called by a dispatcher
type Queue interface {
	Enqueue(func(context.Context)) error // Adds item to the queue
	Start()                              // Starts the processing of the queue
	Stop()                               // Stops the processing of the queue
}

// QueueRepository is a store of Queue items
type QueueRepository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, string) (Item, error)
	Create(context.Context, Item) error
}

// Item is an item in the QueueRepository
type Item struct {
	ID   string
	Dag  *dag.Node[[]string]
	Kind string
	Name string
	CID  string
}
