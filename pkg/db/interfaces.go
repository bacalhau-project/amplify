package db

import (
	"context"

	"github.com/google/uuid"
)

var _ NodePersistence = (*Queries)(nil)

type NodePersistence interface {
	CreateNodeReturnId(ctx context.Context, arg CreateNodeReturnIdParams) (int32, error)
	GetNodeByID(ctx context.Context, id int32) (GetNodeByIDRow, error)
	CreateEdge(ctx context.Context, arg CreateEdgeParams) error
	CreateIOSpec(ctx context.Context, arg CreateIOSpecParams) error
	GetIOSpecByID(ctx context.Context, id int32) (IoSpec, error)
	CreateStatus(ctx context.Context, arg CreateStatusParams) error
	CreateResult(ctx context.Context, arg CreateResultParams) error
}

type Queue interface {
	CreateQueueItem(ctx context.Context, arg CreateQueueItemParams) error
	GetQueueItemDetail(ctx context.Context, id uuid.UUID) (QueueItem, error)
	ListQueueItems(ctx context.Context) ([]QueueItem, error)
	GetNodesByQueueItemID(ctx context.Context, queueItemID uuid.UUID) ([]Node, error)
}
