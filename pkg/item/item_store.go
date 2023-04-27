package item

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	MaxTime             = time.Unix(1<<63-62135596801, 999999999)
	ErrSortNotSupported = fmt.Errorf("sort parameter not supported")
)

type ListParams struct {
	PageSize   int
	PageNumber int
	Sort       string
}

func NewListParams() ListParams {
	return ListParams{
		PageSize:   10,
		PageNumber: 1,
		Sort:       "-created_at",
	}
}

// ItemStore is an interface to retrieve and store items
type ItemStore interface {
	NewItem(ctx context.Context, params ItemParams) error
	ListItems(ctx context.Context, params ListParams) ([]*Item, error)
	CountItems(ctx context.Context) (int64, error)
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

var sort_map = map[string]string{
	"created_at":     "created_at",
	"meta.submitted": "created_at",
}

func (r *itemStore) ListItems(ctx context.Context, params ListParams) ([]*Item, error) {
	reverse := strings.HasPrefix(params.Sort, "-")
	var ok bool
	params.Sort, ok = sort_map[strings.TrimPrefix(params.Sort, "-")]
	if !ok {
		return nil, ErrSortNotSupported
	}
	dbParams := db.ListQueueItemsParams{
		PageSize:   int32(params.PageSize),
		PageNumber: int32(params.PageNumber),
		Sort:       params.Sort,
		Reverse:    reverse,
	}
	log.Ctx(ctx).Trace().Msgf("Listing items with params %+v", dbParams)
	dbItems, err := r.database.ListQueueItems(ctx, dbParams)
	if err != nil {
		return nil, err
	}
	log.Ctx(ctx).Trace().Msgf("Found %d items", len(dbItems))
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

func (r *itemStore) CountItems(ctx context.Context) (int64, error) {
	return r.database.CountQueueItems(ctx)
}

func (r *itemStore) GetItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	dbItem, err := r.database.GetQueueItemDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.buildItemFromDBItem(ctx, dbItem)
}

func (r *itemStore) SetResultMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	for k, v := range metadata {
		err := r.database.CreateResultMetadata(ctx, db.CreateResultMetadataParams{
			QueueItemID: id,
			Type:        k,
			Value:       v,
		})
		if err != nil {
			return err
		}
	}
	return nil
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
