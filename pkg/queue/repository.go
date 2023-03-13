package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/bacalhau-project/amplify/pkg/composite"
	"github.com/bacalhau-project/amplify/pkg/dag"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)

type Item struct {
	ID   string
	Dag  *dag.Node[*composite.Composite]
	Kind string
	Name string
	CID  string
}

type QueueRepository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, string) (Item, error)
	Create(context.Context, Item) error
}

// queueRepository is the implementation of QueueRepository
type queueRepository struct {
	*sync.RWMutex
	store map[string]Item
	queue Queue
}

// NewQueueRepository creates a repository that returns queue information
func NewQueueRepository(queue Queue) QueueRepository {
	return &queueRepository{
		RWMutex: &sync.RWMutex{},
		store:   make(map[string]Item),
		queue:   queue,
	}
}

func (r *queueRepository) List(ctx context.Context) ([]Item, error) {
	list := make([]Item, 0, len(r.store))
	for _, i := range r.store {
		list = append(list, i)
	}
	return list, nil
}

func (r *queueRepository) Get(ctx context.Context, id string) (Item, error) {
	i, ok := r.store[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return i, nil
}

func (r *queueRepository) Create(ctx context.Context, req Item) error {
	if _, ok := r.store[req.ID]; ok {
		return ErrAlreadyExists
	}
	r.store[req.ID] = req
	err := r.queue.Enqueue(req.Dag.Execute)
	if err != nil {
		return err
	}
	return nil
}
