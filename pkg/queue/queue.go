package queue

import (
	"context"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
)

// Queue encapsulates the Enqueue method that will be called by a dispatcher
type Queue interface {
	Enqueue(func(context.Context)) error // Adds item to the queue
	Start()                              // Starts the processing of the queue
	Stop()                               // Stops the processing of the queue
}

// QueueRepository is a store of Queue items
type QueueRepository interface {
	List(context.Context) ([]*Item, error)
	Get(context.Context, uuid.UUID) (*Item, error)
	Create(context.Context, Item) error
}

type ItemMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

// Item is an item in the QueueRepository
type Item struct {
	ID        uuid.UUID
	RootNodes []dag.Node[dag.IOSpec]
	CID       string
	Metadata  ItemMetadata
}
