package item

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
)

var ErrNotFound error = errors.New(`item not found in the queue`)
var ErrAlreadyExists error = errors.New(`item already exists in the queue`)
var ErrItemNoID error = errors.New(`item must have ID`)

type inMemStore struct {
	mu          sync.RWMutex
	store       map[uuid.UUID]*Item
	nodeFactory dag.NodeFactory[dag.IOSpec]
}

func NewInMemItemStore(nodeFactory dag.NodeFactory[dag.IOSpec]) ItemStore {
	return &inMemStore{
		store:       make(map[uuid.UUID]*Item),
		nodeFactory: nodeFactory,
	}
}

func (r *inMemStore) NewItem(ctx context.Context, req ItemParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[req.ID] = &Item{
		ID:  req.ID,
		CID: req.CID,
		Metadata: ItemMetadata{
			CreatedAt: time.Now(),
		},
	}
	return nil
}

func (r *inMemStore) SetNodes(ctx context.Context, id uuid.UUID, nodes []dag.Node[dag.IOSpec]) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, err := r.getItemUnsafe(ctx, id)
	if err != nil {
		return err
	}
	i.RootNodes = nodes
	return nil
}

func (r *inMemStore) ListItems(ctx context.Context) ([]*Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Item, 0, len(r.store))
	for _, i := range r.store {
		list = append(list, i)
	}
	return list, nil
}

func (r *inMemStore) GetItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getItemUnsafe(ctx, id)
}

func (r *inMemStore) getItemUnsafe(ctx context.Context, id uuid.UUID) (*Item, error) {
	i, ok := r.store[id]
	if !ok {
		return nil, ErrNotFound
	}
	return i, nil
}

func (r *inMemStore) GetAllNodes(ctx context.Context, id uuid.UUID) ([]dag.Node[dag.IOSpec], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, err := r.getItemUnsafe(ctx, id)
	if err != nil {
		return nil, err
	}
	return i.RootNodes, nil
}
