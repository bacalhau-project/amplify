package item

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func Test_QueueRepository_Create(t *testing.T) {
	persistence := db.NewInMemDB()
	itemStore := newMockItemStore(persistence)
	executor, _ := dag.NewNodeExecutor[dag.IOSpec](context.Background(), nil)
	repo, err := NewQueueRepository(itemStore, queue.NewMockQueue(), task.NewMockTaskFactory(persistence), executor)
	assert.NilError(t, err)
	tests := []struct {
		name    string
		args    ItemParams
		wantErr bool
	}{
		{"ok", ItemParams{ID: uuid.New(), CID: "cid"}, false},
		{"missing_cid", ItemParams{ID: uuid.New()}, true},
		{"missing_id", ItemParams{CID: "cid"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := repo.Create(context.Background(), tt.args); (err != nil) != tt.wantErr {
				t.Errorf("queueRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_QueueRepository_Get(t *testing.T) {
	persistence := db.NewInMemDB()
	itemStore := newMockItemStore(persistence)
	executor, _ := dag.NewNodeExecutor[dag.IOSpec](context.Background(), nil)
	repo, err := NewQueueRepository(itemStore, queue.NewMockQueue(), task.NewMockTaskFactory(persistence), executor)
	assert.NilError(t, err)
	id := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id, CID: "cid"})
	assert.NilError(t, err)
	i, err := repo.Get(context.Background(), id)
	assert.NilError(t, err)
	assert.Equal(t, i.ID, id)
	assert.Equal(t, len(i.RootNodes), 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		i, err := repo.Get(context.Background(), id)
		assert.NilError(t, err)
		if !i.Metadata.EndedAt.IsZero() || ctx.Err() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	i, err = repo.Get(context.Background(), id)
	assert.NilError(t, err)
	assert.Equal(t, i.Metadata.EndedAt.IsZero(), false)
}

func Test_QueueRepository_List(t *testing.T) {
	persistence := db.NewInMemDB()
	itemStore := newMockItemStore(persistence)
	executor, _ := dag.NewNodeExecutor[dag.IOSpec](context.Background(), nil)
	repo, err := NewQueueRepository(itemStore, queue.NewMockQueue(), task.NewMockTaskFactory(persistence), executor)
	assert.NilError(t, err)
	id1 := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id1, CID: "cid"})
	assert.NilError(t, err)
	id2 := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id2, CID: "cid"})
	assert.NilError(t, err)

	l, err := repo.List(context.Background(), NewListParams())
	assert.NilError(t, err)
	assert.Equal(t, len(l), 2)
	for _, i := range l {
		assert.Equal(t, len(i.RootNodes), 1)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for {
			i, err := repo.Get(ctx, i.ID)
			assert.NilError(t, err)
			if !i.Metadata.EndedAt.IsZero() || ctx.Err() != nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		i, err = repo.Get(context.Background(), i.ID)
		assert.NilError(t, err)
		assert.Equal(t, i.Metadata.EndedAt.IsZero(), false)
	}
}

// func Test_StoreMetadata(t *testing.T) {
// 	persistence := db.NewInMemDB()
// 	itemStore := newMockItemStore(persistence)
// 	repo, err := NewQueueRepository(itemStore, queue.NewMockQueue(), task.NewMockTaskFactory(persistence))
// 	assert.NilError(t, err)
// 	id := uuid.New()
// 	err = repo.Create(context.Background(), ItemParams{ID: id, CID: "cid"})
// 	assert.NilError(t, err)
// 	persistence.QueryTopResultsByKey(context.Background()...
// }

func newMockItemStore(persistence db.Persistence) ItemStore {
	nodeStore, _ := dag.NewNodeStore(context.Background(), persistence, dag.NewInMemWorkRepository[dag.IOSpec]())
	s, _ := NewItemStore(
		context.Background(),
		persistence,
		nodeStore,
	)
	return s
}
