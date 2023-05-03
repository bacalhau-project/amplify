package db

import (
	"context"

	"github.com/google/uuid"
)

type Persistence interface {
	NodePersistence
	Queue
	Analytics
}

type NodePersistence interface {
	CreateAndReturnNode(ctx context.Context, arg CreateAndReturnNodeParams) (Node, error)
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
	ListQueueItems(ctx context.Context, arg ListQueueItemsParams) ([]QueueItem, error)
	CountQueueItems(ctx context.Context) (int64, error)
	GetNodesByQueueItemID(ctx context.Context, queueItemID uuid.UUID) ([]Node, error)
	CreateResultMetadata(ctx context.Context, arg CreateResultMetadataParams) error
}

type Analytics interface {
	QueryTopResultsByKey(ctx context.Context, arg QueryTopResultsByKeyParams) ([]QueryTopResultsByKeyRow, error)
	CreateResultMetadata(ctx context.Context, arg CreateResultMetadataParams) error
	CountQueryTopResultsByKey(ctx context.Context, key string) (int64, error)
	QueryMostRecentResultsByKey(ctx context.Context, arg QueryMostRecentResultsByKeyParams) ([]QueryMostRecentResultsByKeyRow, error)
}
