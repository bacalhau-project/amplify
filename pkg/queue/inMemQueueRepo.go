package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)
var ErrItemNoID error = errors.New(`item must have ID`)

// inMemQueueRepo is the implementation of QueueRepository
type inMemQueueRepo struct {
	*sync.Mutex
	store map[uuid.UUID]*Item
	queue Queue
}

// NewQueueRepository creates a repository that returns queue information
func NewQueueRepository(queue Queue) QueueRepository {
	return &inMemQueueRepo{
		Mutex: &sync.Mutex{},
		store: make(map[uuid.UUID]*Item),
		queue: queue,
	}
}

func (r *inMemQueueRepo) List(ctx context.Context) ([]*Item, error) {
	r.Lock()
	defer r.Unlock()
	list := make([]*Item, 0, len(r.store))
	for _, i := range r.store {
		r.updateStartStopTime(ctx, i.ID)
	}
	for _, i := range r.store {
		list = append(list, i)
	}
	return list, nil
}

func (r *inMemQueueRepo) Get(ctx context.Context, id uuid.UUID) (*Item, error) {
	r.Lock()
	defer r.Unlock()
	i, ok := r.store[id]
	if !ok {
		return nil, ErrNotFound
	}
	r.updateStartStopTime(ctx, id)
	return i, nil
}

// TODO: This is really bad. We should use a channel to set this. Easy to forget too.
func (r *inMemQueueRepo) updateStartStopTime(ctx context.Context, id uuid.UUID) {
	i, ok := r.store[id]
	if !ok {
		log.Error().Msgf("item %s not found in the queue", id)
		return
	}
	if i.Metadata.StartedAt.IsZero() {
		t, err := dag.GetDagStartTime(ctx, i.RootNodes)
		if err != nil {
			return
		}
		i.Metadata.StartedAt = t
	}
	if i.Metadata.EndedAt.IsZero() {
		t, err := dag.GetEndTimeIfDagComplete(ctx, i.RootNodes)
		if err != nil {
			return
		}
		i.Metadata.EndedAt = t
	}
}

func (r *inMemQueueRepo) Create(ctx context.Context, req Item) error {
	if req.Metadata.CreatedAt.IsZero() {
		req.Metadata.CreatedAt = time.Now()
	}
	r.Lock()
	defer r.Unlock()
	if _, ok := r.store[req.ID]; ok {
		return ErrAlreadyExists
	}
	r.store[req.ID] = &req
	for _, d := range req.RootNodes {
		err := r.queue.Enqueue(func(ctx context.Context) {
			dag.Execute(ctx, d)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
