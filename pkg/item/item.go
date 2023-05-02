package item

import (
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID
	RootNodes []dag.Node[dag.IOSpec]
	CID       string
	Metadata  ItemMetadata
}

type ItemMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

type ItemParams struct {
	ID       uuid.UUID
	CID      string
	Priority bool
}
