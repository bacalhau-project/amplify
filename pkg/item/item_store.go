package item

import (
	"context"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	MaxTime = time.Unix(1<<63-62135596801, 999999999)
)

type PaginationParams struct {
	Limit         int
	CreatedBefore time.Time
	CreatedAfter  time.Time
	Reverse       bool
}

// ItemStore is an interface to retrieve and store items
type ItemStore interface {
	NewItem(ctx context.Context, params ItemParams) error
	ListItems(ctx context.Context, params PaginationParams) ([]*Item, error)
	GetItem(ctx context.Context, id uuid.UUID) (*Item, error)
}

type itemStore struct {
	database  db.Queue
	nodeStore dag.NodeStore[dag.IOSpec]
}

func NewItemStore(ctx context.Context, database db.Queue, nodeStore dag.NodeStore[dag.IOSpec]) (ItemStore, error) {
	return &itemStore{
		database:  database,
		nodeStore: nodeStore,
	}, nil
}

func (r *itemStore) NewItem(ctx context.Context, req ItemParams) error {
	return r.database.CreateQueueItem(ctx, db.CreateQueueItemParams{
		ID:        req.ID,
		Inputs:    []string{req.CID},
		CreatedAt: time.Now(),
	})
}

func (r *itemStore) ListItems(ctx context.Context, params PaginationParams) ([]*Item, error) {
	if params.CreatedBefore.IsZero() {
		params.CreatedBefore = time.Now()
	}
	if params.Limit == 0 {
		params.Limit = 10
	}
	log.Ctx(ctx).Trace().Msgf("Listing items with params %+v", params)
	dbItems, err := r.database.ListQueueItems(ctx, db.ListQueueItemsParams{
		Limit:         int32(params.Limit),
		Createdbefore: params.CreatedBefore,
		Createdafter:  params.CreatedAfter,
		Reverse:       params.Reverse,
	})
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

func (r *itemStore) GetItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	dbItem, err := r.database.GetQueueItemDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.buildItemFromDBItem(ctx, dbItem)
}

func (r *itemStore) buildItemFromDBItem(ctx context.Context, dbItem db.QueueItem) (*Item, error) {
	d, err := r.getAllNodes(ctx, dbItem.ID)
	if err != nil {
		return nil, err
	}
	d, err = dag.FilterForRootNodes(ctx, d)
	if err != nil {
		return nil, err
	}
	return &Item{
		ID: dbItem.ID,
		Metadata: ItemMetadata{
			CreatedAt: dbItem.CreatedAt,
		},
		CID:       dbItem.Inputs[0],
		RootNodes: d,
	}, nil
}

func (r *itemStore) getAllNodes(ctx context.Context, id uuid.UUID) ([]dag.Node[dag.IOSpec], error) {
	nodes, err := r.database.GetNodesByQueueItemID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := make([]dag.Node[dag.IOSpec], 0, len(nodes))
	for _, n := range nodes {
		nodeI, err := r.nodeStore.GetNode(ctx, n.ID)
		if err != nil {
			return nil, err
		}
		d = append(d, nodeI)
	}
	return d, nil
}
