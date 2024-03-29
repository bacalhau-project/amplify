package item

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidRequestID  = fmt.Errorf("invalid request id")
	ErrInvalidRequestCID = fmt.Errorf("invalid request cid")
)

// QueueRepository is a repository of Queue items
type QueueRepository interface {
	List(context.Context, ListParams) ([]*Item, error)
	Count(context.Context) (int64, error)
	Get(context.Context, uuid.UUID) (*Item, error)
	Create(context.Context, ItemParams) error
}

type queueRepository struct {
	repo           ItemStore
	tf             task.TaskFactory
	priorityQueue  queue.Queue
	secondaryQueue queue.Queue
	nodeExecutor   dag.NodeExecutor[dag.IOSpec]
}

func NewQueueRepository(repo ItemStore, priorityQueue queue.Queue, secondaryQueue queue.Queue, taskFactory task.TaskFactory, nodeExecutor dag.NodeExecutor[dag.IOSpec]) (QueueRepository, error) {
	if repo == nil || taskFactory == nil || secondaryQueue == nil || priorityQueue == nil || nodeExecutor == nil {
		return nil, fmt.Errorf("missing dependencies")
	}
	return &queueRepository{
		repo:           repo,
		tf:             taskFactory,
		priorityQueue:  priorityQueue,
		secondaryQueue: secondaryQueue,
		nodeExecutor:   nodeExecutor,
	}, nil
}

func (r *queueRepository) Create(ctx context.Context, req ItemParams) error {
	if req.ID == uuid.Nil {
		return ErrInvalidRequestID
	}
	if req.CID == "" {
		return ErrInvalidRequestCID
	}
	q := r.secondaryQueue
	if req.Priority {
		q = r.priorityQueue
	}
	if q.IsFull() {
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
	for _, node := range dags {
		err := q.Enqueue(func(ctx context.Context) {
			// Execute the node
			r.nodeExecutor.Execute(ctx, req.ID, node)
			// Wait until all the nodes in the dag are completed
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					finished, err := dag.AllNodesFinished(ctx, []dag.Node[dag.IOSpec]{node})
					if err != nil {
						log.Ctx(ctx).Error().Err(err).Msg("error checking if all nodes are finished")
						return
					}
					if finished {
						log.Ctx(ctx).Info().Msg("all nodes are finished")
						return
					}
				}
			}
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

func (r *queueRepository) List(ctx context.Context, params ListParams) ([]*Item, error) {
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

func (r *queueRepository) Count(ctx context.Context) (int64, error) {
	return r.repo.CountItems(ctx)
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
