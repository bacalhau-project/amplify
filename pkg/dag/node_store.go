package dag

import (
	"context"

	"github.com/bacalhau-project/amplify/pkg/db"
)

type NodeStore[T any] interface {
	NewNode(context.Context, NodeSpec[T]) (Node[T], error)
	GetNode(context.Context, int32) (Node[T], error)
}

func NewNodeStore(ctx context.Context, p db.NodePersistence, wr WorkRepository[IOSpec]) (NodeStore[IOSpec], error) {
	return &nodeStore{
		Persistence:    p,
		WorkRepository: wr,
	}, nil
}

type nodeStore struct {
	Persistence    db.NodePersistence
	WorkRepository WorkRepository[IOSpec]
}

func (f *nodeStore) NewNode(ctx context.Context, n NodeSpec[IOSpec]) (Node[IOSpec], error) {
	return NewNode(ctx, f.Persistence, f.WorkRepository, n)
}

func (f *nodeStore) GetNode(ctx context.Context, id int32) (Node[IOSpec], error) {
	return nodeNodeWithID(f.Persistence, f.WorkRepository, id), nil
}
