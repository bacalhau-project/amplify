package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)
var ErrItemNoID error = errors.New(`item must have ID`)

// inMemQueueRepo is the implementation of QueueRepository
type inMemQueueRepo struct {
	*sync.Mutex
	store map[string]*Item
	queue Queue
}

// NewQueueRepository creates a repository that returns queue information
func NewQueueRepository(queue Queue) QueueRepository {
	return &inMemQueueRepo{
		Mutex: &sync.Mutex{},
		store: make(map[string]*Item),
		queue: queue,
	}
}

func (r *inMemQueueRepo) List(ctx context.Context) ([]*Item, error) {
	r.Lock()
	defer r.Unlock()
	list := make([]*Item, 0, len(r.store))
	for _, i := range r.store {
		r.updateStartStopTime(i.ID)
	}
	for _, i := range r.store {
		list = append(list, i)
	}
	return list, nil
}

func (r *inMemQueueRepo) Get(ctx context.Context, id string) (*Item, error) {
	r.Lock()
	defer r.Unlock()
	i, ok := r.store[id]
	if !ok {
		return nil, ErrNotFound
	}
	r.updateStartStopTime(id)
	return i, nil
}

// TODO: This is really bad. We should use a channel to set this
func (r *inMemQueueRepo) updateStartStopTime(id string) {
	i := r.store[id]
	if i.Metadata.StartedAt.IsZero() {
		for _, d := range i.Dag {
			if !d.Meta().StartedAt.IsZero() {
				i.Metadata.StartedAt = d.Meta().StartedAt
				break
			}
		}
	}
	if i.Metadata.EndedAt.IsZero() {
		// All dags must have finished
		var t time.Time
		ok := true
		for _, d := range i.Dag {
			finTime := recurseLastTime(d)
			if finTime.IsZero() {
				ok = false
				break
			}
			if finTime.After(t) {
				t = finTime
			}
		}
		if ok {
			i.Metadata.EndedAt = t
		}
	}
}

func recurseLastTime(d *dag.Node[dag.IOSpec]) time.Time {
	if len(d.Children()) == 0 {
		return d.Meta().EndedAt
	}
	var t time.Time
	for _, c := range d.Children() {
		t = recurseLastTime(c)
	}
	return t
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
	r.store[req.ID] = &req
	for _, d := range req.Dag {
		err := r.queue.Enqueue(d.Execute)
		if err != nil {
			return err
		}
	}
	return nil
}
