package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
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
	r, err := NewPostgresQueueRepository(connStr, queue)
	assert.NilError(t, err)

	// Create a DAG
	rootNode := dag.NewNode("root",
		nilFunc,
	)
	childNode := dag.NewNode("child",
		nilFunc,
	)
	rootNode.AddChild(childNode)
	id := uuid.New().String()
	cid := "Qm123"
	err = r.Create(ctx, Item{ID: id, Dag: []*dag.Node[dag.IOSpec]{
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

func nilFunc(ctx context.Context, inputs []dag.IOSpec, statusChan chan dag.NodeStatus) []dag.IOSpec {
	defer close(statusChan)
	statusChan <- dag.NodeStatus{
		Status: "test",
	}
	return nil
}
