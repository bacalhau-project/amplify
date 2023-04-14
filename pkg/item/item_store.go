package item

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
)

// ItemStore is an interface to retrieve and store items
type ItemStore interface {
	NewItem(ctx context.Context, params ItemParams) error
	ListItems(ctx context.Context) ([]*Item, error)
	GetItem(ctx context.Context, id uuid.UUID) (*Item, error)
	GetAllNodes(ctx context.Context, id uuid.UUID) ([]dag.Node[dag.IOSpec], error)
	SetNodes(ctx context.Context, id uuid.UUID, nodes []dag.Node[dag.IOSpec]) error
}

func NewMockItemStore() ItemStore {
	return NewInMemItemStore(dag.NewInMemNodeFactory(dag.NewInMemWorkRepository[dag.IOSpec]()))
}
