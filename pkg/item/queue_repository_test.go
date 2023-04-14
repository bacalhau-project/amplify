package item

import (
	"context"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func Test_QueueRepository_Create(t *testing.T) {
	repo, err := NewQueueRepository(NewMockItemStore(), queue.NewMockQueue(), task.NewMockTaskFactory())
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
	repo, err := NewQueueRepository(NewMockItemStore(), queue.NewMockQueue(), task.NewMockTaskFactory())
	assert.NilError(t, err)
	id := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id, CID: "cid"})
	assert.NilError(t, err)

	// tests := []struct {
	// 	name    string
	// 	args    uuid.UUID
	// 	wantErr bool
	// }{
	// 	{"ok", id, false},
	// 	{"missing", uuid.New(), true},
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		if _, err := repo.Get(context.Background(), tt.args); (err != nil) != tt.wantErr {
	// 			t.Errorf("queueRepository.Get() error = %v, wantErr %v", err, tt.wantErr)
	// 		}
	// 	})
	// }
	i, err := repo.Get(context.Background(), id)
	assert.NilError(t, err)
	assert.Equal(t, i.ID, id)
	assert.Equal(t, len(i.RootNodes), 1)
	assert.Equal(t, i.Metadata.EndedAt.IsZero(), false)
}

func Test_QueueRepository_List(t *testing.T) {
	repo, err := NewQueueRepository(NewMockItemStore(), queue.NewMockQueue(), task.NewMockTaskFactory())
	assert.NilError(t, err)
	id1 := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id1, CID: "cid"})
	assert.NilError(t, err)
	id2 := uuid.New()
	err = repo.Create(context.Background(), ItemParams{ID: id2, CID: "cid"})
	assert.NilError(t, err)

	l, err := repo.List(context.Background())
	assert.NilError(t, err)
	assert.Equal(t, len(l), 2)
	for _, i := range l {
		assert.Equal(t, len(i.RootNodes), 1)
		assert.Equal(t, i.Metadata.EndedAt.IsZero(), false)
	}
}
