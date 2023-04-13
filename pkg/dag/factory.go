package dag

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/db"
)

// TODO: The nodes should implement this interface themselves
type NodeFactory[T any] interface {
	NewNode(context.Context, NodeSpec[T]) (Node[T], error)
	GetNode(context.Context, int32) (Node[T], error)
}

var _ NodeFactory[IOSpec] = (*PostgresNodeFactory)(nil)

type PostgresNodeFactory struct {
	Persistence    db.NodePersistence
	WorkRepository WorkRepository[IOSpec]
}

func (f *PostgresNodeFactory) NewNode(ctx context.Context, n NodeSpec[IOSpec]) (Node[IOSpec], error) {
	return NewPostgresNode(ctx, f.Persistence, f.WorkRepository, n)
}

func (f *PostgresNodeFactory) GetNode(ctx context.Context, id int32) (Node[IOSpec], error) {
	return newPostgresNodeWithID(f.Persistence, f.WorkRepository, id), nil
}

var _ NodeFactory[IOSpec] = (*InMemNodeFactory[IOSpec])(nil)

type InMemNodeFactory[T any] struct {
	WorkRepository WorkRepository[T]
	store          map[int32]Node[T]
}

func (f *InMemNodeFactory[T]) NewNode(ctx context.Context, n NodeSpec[T]) (Node[T], error) {
	node, err := NewInMemoryNode(ctx, f.WorkRepository, n)
	if err != nil {
		return nil, err
	}
	f.store[node.ID()] = node
	return node, nil
}

func (f *InMemNodeFactory[T]) GetNode(ctx context.Context, id int32) (Node[T], error) {
	node, ok := f.store[id]
	if !ok {
		return nil, fmt.Errorf("node with id %d not found", id)
	}
	return node, nil
}
