package queue

import (
	"context"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
)

var (
	MaxTime = time.Unix(1<<63-62135596801, 999999999)
)

type postgresQueueRepository struct {
	*sync.Mutex
	queries     db.Queue
	nodeFactory dag.NodeFactory[dag.IOSpec]
	queue       Queue
}

func NewPostgresQueueRepository(postgresDb db.Queue, nodeFactory dag.NodeFactory[dag.IOSpec], queue Queue) (QueueRepository, error) {
	return &postgresQueueRepository{
		Mutex:       &sync.Mutex{},
		nodeFactory: nodeFactory,
		queries:     postgresDb,
		queue:       queue,
	}, nil
}

func (r *postgresQueueRepository) Create(ctx context.Context, req Item) error {
	r.Lock()
	defer r.Unlock()
	_, err := r.queries.GetQueueItemDetail(ctx, req.ID)
	if err == nil {
		return ErrAlreadyExists
	}
	err = r.queries.CreateQueueItem(ctx, db.CreateQueueItemParams{
		ID:        req.ID,
		Inputs:    []string{req.CID},
		CreatedAt: req.Metadata.CreatedAt,
	})
	if err != nil {
		return err
	}
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

func (r *postgresQueueRepository) Get(ctx context.Context, id uuid.UUID) (*Item, error) {
	r.Lock()
	defer r.Unlock()
	itemDB, err := r.queries.GetQueueItemDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.buildItemFromDBItem(ctx, itemDB)
}

func (r *postgresQueueRepository) List(ctx context.Context) ([]*Item, error) {
	r.Lock()
	defer r.Unlock()
	dbItems, err := r.queries.ListQueueItems(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*Item, 0, len(dbItems))
	for _, i := range dbItems {
		item, err := r.buildItemFromDBItem(ctx, i)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}

func (r *postgresQueueRepository) buildItemFromDBItem(ctx context.Context, dbItem db.QueueItem) (*Item, error) {
	item := &Item{
		ID: dbItem.ID,
		Metadata: ItemMetadata{
			CreatedAt: dbItem.CreatedAt,
		},
		CID: dbItem.Inputs[0],
	}
	nodes, err := r.queries.GetNodesByQueueItemID(ctx, dbItem.ID)
	if err != nil {
		return nil, err
	}
	d := make([]dag.Node[dag.IOSpec], 0, len(nodes))
	for _, n := range nodes {
		nodeI, err := r.nodeFactory.GetNode(ctx, n.ID)
		if err != nil {
			return nil, err
		}
		d = append(d, nodeI)
	}
	d, err = dag.FilterForRootNodes(ctx, d)
	if err != nil {
		return nil, err
	}
	item.RootNodes = d
	startedAt, err := dag.GetDagStartTime(ctx, d)
	if err != nil {
		return nil, err
	}
	item.Metadata.StartedAt = startedAt
	endedAt, err := dag.GetEndTimeIfDagComplete(ctx, d)
	if err != nil {
		return nil, err
	}
	item.Metadata.EndedAt = endedAt
	return item, nil
}
