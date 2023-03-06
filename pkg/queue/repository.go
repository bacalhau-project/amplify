package queue

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)

// Item is external representation of an item in the queue
// This item is likely to be serialised into a persistence store, so go easy on
// the types.
type Item struct {
	ID        string
	Kind      string
	Name      string
	CID       string
	Submitted time.Time
	Started   time.Time
	Ended     time.Time
}

type CreateRequest struct {
	ID   string
	Kind string
	Name string
	CID  string
	Task Callable
}

type QueueRepository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, string) (Item, error)
	Create(context.Context, CreateRequest) error
	SetStartTime(context.Context, string, time.Time) error
	SetEndTime(context.Context, string, time.Time) error
}

// queueRepository is the implementation of QueueRepository
type queueRepository struct {
	*sync.RWMutex
	store map[string]Item
	queue Queuer
}

// NewQueueRepository creates a repository that returns queue information
func NewQueueRepository(queue Queuer) QueueRepository {
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

func (r *queueRepository) Create(ctx context.Context, req CreateRequest) error {
	if _, ok := r.store[req.ID]; ok {
		return ErrAlreadyExists
	}
	r.store[req.ID] = Item{
		ID:        req.ID,
		Kind:      req.Kind,
		Name:      req.Name,
		CID:       req.CID,
		Submitted: time.Now(),
	}
	err := r.queue.Enqueue(r, req.ID, req.Task)
	if err != nil {
		return err
	}
	return nil
}

func (r *queueRepository) SetStartTime(ctx context.Context, id string, t time.Time) error {
	r.Lock()
	defer r.Unlock()
	i, ok := r.store[id]
	if !ok {
		return ErrNotFound
	}
	i.Started = t
	r.store[id] = i
	return nil
}

func (r *queueRepository) SetEndTime(ctx context.Context, id string, t time.Time) error {
	r.Lock()
	defer r.Unlock()
	i, ok := r.store[id]
	if !ok {
		return ErrNotFound
	}
	i.Ended = t
	r.store[id] = i
	return nil
}
