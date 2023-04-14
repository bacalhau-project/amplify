package item

import (
	"context"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
)

var (
	MaxTime = time.Unix(1<<63-62135596801, 999999999)
)

type postgresItemStore struct {
	database    db.Queue
	nodeFactory dag.NodeFactory[dag.IOSpec]
}

func NewPostgresItemStore(ctx context.Context, database db.Queue, nodeFactory dag.NodeFactory[dag.IOSpec]) (ItemStore, error) {
	return &postgresItemStore{
		database:    database,
		nodeFactory: nodeFactory,
	}, nil
}

func (r *postgresItemStore) NewItem(ctx context.Context, req ItemParams) error {
	return r.database.CreateQueueItem(ctx, db.CreateQueueItemParams{
		ID:        req.ID,
		Inputs:    []string{req.CID},
		CreatedAt: time.Now(),
	})
}

// TODO: This is rubbish, improve
func (*postgresItemStore) SetNodes(ctx context.Context, id uuid.UUID, nodes []dag.Node[dag.IOSpec]) error {
	return nil
}

func (r *postgresItemStore) ListItems(ctx context.Context) ([]*Item, error) {
	dbItems, err := r.database.ListQueueItems(ctx)
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

func (r *postgresItemStore) GetItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	dbItem, err := r.database.GetQueueItemDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.buildItemFromDBItem(ctx, dbItem)
}

func (r *postgresItemStore) GetAllNodes(ctx context.Context, id uuid.UUID) ([]dag.Node[dag.IOSpec], error) {
	nodes, err := r.database.GetNodesByQueueItemID(ctx, id)
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
	return d, nil
}

func (r *postgresItemStore) buildItemFromDBItem(ctx context.Context, dbItem db.QueueItem) (*Item, error) {
	return &Item{
		ID: dbItem.ID,
		Metadata: ItemMetadata{
			CreatedAt: dbItem.CreatedAt,
		},
		CID: dbItem.Inputs[0],
	}, nil
}
