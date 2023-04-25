package item

import (
	"context"
	"encoding/json"
	"fmt"

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
	for _, node := range dags {
		err := r.queue.Enqueue(func(ctx context.Context) {
			// Execute the node
			dag.Execute(ctx, node)

			// TODO: Probably better if we do this in the node execution itself
			// then the update would be much quicker.
			// Parse all the stdouts and persist to DB
			err = r.parseStdoutForAllChildren(ctx, req.ID, node)
			if err != nil {
				log.Error().Err(err).Msg("error parsing stdout")
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

func (r *queueRepository) parseStdoutForAllChildren(ctx context.Context, id uuid.UUID, node dag.Node[dag.IOSpec]) error {
	// Get the results from the execution
	n, err := node.Get(ctx)
	if err != nil {
		return err
	}

	// Parse the stdout as a map and write to the DB
	log.Ctx(ctx).Trace().Msgf("node %s parsing and storing results: %s", n.Name, n.Results.StdOut)
	var resultMap map[string]string
	err = json.Unmarshal([]byte(n.Results.StdOut), &resultMap)
	if err != nil {
		log.Ctx(ctx).Trace().Err(err).Msg("failed to parse stdout json as map, skipping")
	} else {
		err = r.repo.SetResultMetadata(ctx, id, resultMap)
		if err != nil {
			return err
		}
	}
	for _, child := range n.Children {
		err = r.parseStdoutForAllChildren(ctx, id, child)
		if err != nil {
			return err
		}
	}
	return nil
}
