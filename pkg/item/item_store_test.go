//go:build integration || !unit

package item

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestPostgresIntegration(t *testing.T) {
	connStr := os.Getenv("AMPLIFY_DB_URI")
	if connStr == "" {
		t.Skip("set AMPLIFY_DB_URI to run this test")
	}
	ctx := context.Background()
	persistence, err := db.NewPostgresDB(connStr)
	assert.NilError(t, err)
	wr := dag.NewInMemWorkRepository[dag.IOSpec]()
	nodeStore, err := dag.NewNodeStore(ctx, persistence, wr)
	assert.NilError(t, err)
	r, err := NewItemStore(ctx, persistence, nodeStore)
	assert.NilError(t, err)

	id := uuid.New()

	// Create a DAG
	f, err := dag.NewNodeStore(ctx,
		persistence,
		wr,
	)
	assert.NilError(t, err)
	cid := "Qm123"
	err = r.NewItem(ctx, ItemParams{
		ID:  id,
		CID: cid,
	})
	assert.NilError(t, err)

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

	// Make sure pertinent info is stored
	itemDetail, err := r.GetItem(ctx, id)
	assert.NilError(t, err)
	assert.Equal(t, itemDetail.ID, id)
	assert.Equal(t, itemDetail.Metadata.CreatedAt.IsZero(), false)
	assert.Equal(t, itemDetail.Metadata.StartedAt.IsZero(), true)
	assert.Equal(t, itemDetail.Metadata.EndedAt.IsZero(), true)
	assert.Equal(t, itemDetail.CID, cid)

	// List items
	items, err := r.ListItems(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(items) > 0)
}

func TestPostgresPerformance(t *testing.T) {
	connStr := os.Getenv("AMPLIFY_DB_URI")
	if connStr == "" {
		t.Skip("set AMPLIFY_DB_URI to run this test")
	}
	ctx := context.Background()
	persistence, err := db.NewPostgresDB(connStr)
	assert.NilError(t, err)
	wr := dag.NewInMemWorkRepository[dag.IOSpec]()
	nodeStore, err := dag.NewNodeStore(ctx, persistence, wr)
	assert.NilError(t, err)
	r, err := NewItemStore(ctx, persistence, nodeStore)
	assert.NilError(t, err)

	// Create a DAG
	f, err := dag.NewNodeStore(ctx,
		persistence,
		wr,
	)
	assert.NilError(t, err)
	cid := "Qm123"

	// Create thousands of items
	start := time.Now()
	for i := 0; i < 1000; i++ {
		id := uuid.New()
		err = r.NewItem(ctx, ItemParams{
			ID:  id,
			CID: cid,
		})
		assert.NilError(t, err)

		rootNode, err := f.NewNode(ctx, dag.NodeSpec[dag.IOSpec]{
			OwnerID: id,
			Name:    "root",
			Work:    dag.NilFunc,
		})
		assert.NilError(t, err)
		err = rootNode.AddInput(ctx, dag.NewIOSpec("input", "hello", "cid123", "/root", true, "text/plain"))
		assert.NilError(t, err)
		child, err := f.NewNode(ctx, dag.NodeSpec[dag.IOSpec]{
			OwnerID: id,
			Name:    "child",
			Work:    dag.NilFunc,
		})
		assert.NilError(t, err)
		err = rootNode.AddParentChildRelationship(ctx, child)
		assert.NilError(t, err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Creating took %s\n", elapsed)

	// Test getting the node repeatedly
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err = f.GetNode(ctx, int32(1))
		assert.NilError(t, err)
	}
	elapsed = time.Since(start)
	fmt.Printf("Getting took %s\n", elapsed)
}

var _ queue.Queue = &testQueue{}

type testQueue struct {
	QueueCount int
}

func (t *testQueue) Enqueue(f func(context.Context)) error {
	t.QueueCount += 1
	f(context.Background())
	return nil
}

func (*testQueue) Start() {
}

func (*testQueue) Stop() {
}
