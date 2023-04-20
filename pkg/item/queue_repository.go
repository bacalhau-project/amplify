package item

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/google/uuid"
)

var (
	ErrInvalidRequestID  = fmt.Errorf("invalid request id")
	ErrInvalidRequestCID = fmt.Errorf("invalid request cid")
)

// QueueRepository is a repository of Queue items
type QueueRepository interface {
	List(context.Context, PaginationParams) ([]*Item, error)
	Get(context.Context, uuid.UUID) (*Item, error)
	Create(context.Context, ItemParams) error
}

type queueRepository struct {
	repo  ItemStore
	tf    task.TaskFactory
	queue queue.Queue
}

func NewQueueRepository(repo ItemStore, queue queue.Queue, taskFactory task.TaskFactory) (QueueRepository, error) {
	if repo == nil || taskFactory == nil || queue == nil {
		return nil, fmt.Errorf("missing dependencies")
	}
	return &queueRepository{
		repo:  repo,
		tf:    taskFactory,
		queue: queue,
	}, nil
}

func (r *queueRepository) Create(ctx context.Context, req ItemParams) error {
	if req.ID == uuid.Nil {
		return ErrInvalidRequestID
	}
	if req.CID == "" {
		return ErrInvalidRequestCID
	}
	if r.queue.IsFull() {
		return queue.ErrQueueFull
	}
	err := r.repo.NewItem(ctx, req)
	if err != nil {
		return err
	}
	dags, err := r.tf.CreateTask(ctx, req.ID, req.CID)
	if err != nil {
		return err
	}
	for _, d := range dags {
		err := r.queue.Enqueue(func(ctx context.Context) {
			dag.Execute(ctx, d)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *queueRepository) Get(ctx context.Context, id uuid.UUID) (*Item, error) {
	item, err := r.repo.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}
	err = r.cleanItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *queueRepository) List(ctx context.Context, params PaginationParams) ([]*Item, error) {
	items, err := r.repo.ListItems(ctx, params)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		err := r.cleanItem(ctx, item)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *queueRepository) cleanItem(ctx context.Context, item *Item) error {
	startedAt, err := dag.GetDagStartTime(ctx, item.RootNodes)
	if err != nil {
		return err
	}
	item.Metadata.StartedAt = startedAt
	endedAt, err := dag.GetEndTimeIfDagComplete(ctx, item.RootNodes)
	if err != nil {
		return err
	}
	item.Metadata.EndedAt = endedAt
	return nil
}