package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestNewQueueRepository(t *testing.T) {
	inMemQueueRepo := NewQueueRepository(&testQueue{})
	assert.Assert(t, inMemQueueRepo != nil)
}

func Test_queueRepository_CreateListGet(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	node, err := dag.NewInMemoryNode(
		ctx,
		dag.NewInMemWorkRepository[dag.IOSpec](),
		dag.NodeSpec[dag.IOSpec]{
			OwnerID: id,
			Work:    dag.NilFunc,
		},
	)
	assert.NilError(t, err)
	item := Item{
		ID:        id,
		RootNodes: []dag.Node[dag.IOSpec]{node},
		Metadata: ItemMetadata{
			CreatedAt: time.Now(),
		},
	}
	inMemQueueRepo := NewQueueRepository(&testQueue{})
	err = inMemQueueRepo.Create(ctx, item)
	assert.NilError(t, err)
	items, err := inMemQueueRepo.List(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(items) == 1)
	assert.Equal(t, items[0].Metadata.CreatedAt.IsZero(), false)
	assert.Equal(t, items[0].Metadata.StartedAt.IsZero(), false)
	assert.Equal(t, items[0].Metadata.EndedAt.IsZero(), false)
}

func Test_queueRepository_Concurrency(t *testing.T) {
	ctx := context.Background()
	q := NewQueueRepository(&testQueue{})
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(3)
		test := uuid.New()
		go func(test uuid.UUID) {
			err := q.Create(ctx, Item{ID: test})
			if err != nil && err != ErrAlreadyExists {
				t.Error(err)
			}
			wg.Done()
		}(test)
		go func() {
			_, err := q.List(ctx)
			if err != nil {
				t.Error(err)
			}
			wg.Done()
		}()
		go func(test uuid.UUID) {
			_, err := q.Get(ctx, test)
			if err != nil && err != ErrNotFound {
				t.Error(err)
			}
			wg.Done()
		}(test)
	}
	wg.Wait()
}

var _ Queue = &testQueue{}

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
