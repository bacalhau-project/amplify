package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)
var ErrItemNoID error = errors.New(`item must have ID`)

// inMemQueueRepo is the implementation of QueueRepository
type inMemQueueRepo struct {
	*sync.RWMutex
	store map[string]Item
	queue Queue
}

// NewQueueRepository creates a repository that returns queue information
func NewQueueRepository(queue Queue) QueueRepository {
	return &inMemQueueRepo{
		RWMutex: &sync.RWMutex{},
		store:   make(map[string]Item),
		queue:   queue,
	}
}

func (r *inMemQueueRepo) List(ctx context.Context) ([]Item, error) {
	r.RLock()
	defer r.RUnlock()
	list := make([]Item, 0, len(r.store))
	for _, i := range r.store {
		list = append(list, i)
	}
	return list, nil
}

func (r *inMemQueueRepo) Get(ctx context.Context, id string) (Item, error) {
	r.RLock()
	defer r.RUnlock()
	i, ok := r.store[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return i, nil
}

func (r *inMemQueueRepo) Create(ctx context.Context, req Item) error {
	r.Lock()
	defer r.Unlock()
	if req.ID == "" {
		return ErrItemNoID
	}
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
