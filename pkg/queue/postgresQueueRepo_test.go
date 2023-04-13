//go:build integration || !unit

package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestPostgresIntegration(t *testing.T) {
	connStr := os.Getenv("AMPLIFY_DB_URI")
	if connStr == "" {
		t.Skip("set AMPLIFY_DB_URI to run this test")
	}
	ctx := context.Background()
	queue := &testQueue{}
	queries, err := db.NewPostgresDB(connStr)
	assert.NilError(t, err)
	wr := dag.NewInMemWorkRepository[dag.IOSpec]()
	nodeFactory := dag.PostgresNodeFactory{
		Persistence:    queries,
		WorkRepository: wr,
	}
	r, err := NewPostgresQueueRepository(queries, &nodeFactory, queue)
	assert.NilError(t, err)

	id := uuid.New()

	// Create a DAG
	f := dag.PostgresNodeFactory{
		Persistence:    queries,
		WorkRepository: wr,
	}

	rootNode, err := f.NewNode(ctx, dag.NodeSpec[dag.IOSpec]{
		OwnerID: id,
		Name:    "root",
		Work:    dag.NilFunc,
	})
	assert.NilError(t, err)
	err = rootNode.AddInput(ctx, dag.NewIOSpec("input", "hello", "cid123", "/root", true, "text/plain"))
	assert.NilError(t, err)
	childNode, err := f.NewNode(ctx, dag.NodeSpec[dag.IOSpec]{
		OwnerID: id,
		Name:    "child",
		Work:    dag.NilFunc,
	})
	assert.NilError(t, err)
	err = rootNode.AddParentChildRelationship(ctx, childNode)
	assert.NilError(t, err)
	cid := "Qm123"
	err = r.Create(ctx, Item{
		ID: id,
		RootNodes: []dag.Node[dag.IOSpec]{
			rootNode,
		},
		Metadata: ItemMetadata{CreatedAt: time.Now()},
		CID:      cid,
	})
	assert.NilError(t, err)
	assert.Equal(t, queue.QueueCount, 1)

	// Make sure pertinent info is stored
	itemDetail, err := r.Get(ctx, id)
	assert.NilError(t, err)
	assert.Equal(t, itemDetail.ID, id)
	assert.Equal(t, itemDetail.Metadata.CreatedAt.IsZero(), false)
	assert.Equal(t, itemDetail.Metadata.StartedAt.IsZero(), false)
	assert.Equal(t, itemDetail.Metadata.EndedAt.IsZero(), false)
	assert.Equal(t, itemDetail.CID, cid)

	// List items
	items, err := r.List(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(items) > 0)
}
