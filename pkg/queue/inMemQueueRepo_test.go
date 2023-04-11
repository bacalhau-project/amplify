package queue

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gotest.tools/assert"
)

func TestNewQueueRepository(t *testing.T) {
	type args struct {
		queue Queue
	}
	tests := []struct {
		name string
		args args
		want QueueRepository
	}{
		{"test", args{queue: &testQueue{}}, &inMemQueueRepo{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewQueueRepository(tt.args.queue)
			assert.Assert(t, got != nil)
		})
	}
}

func Test_queueRepository_List(t *testing.T) {
	type fields struct {
		Mutex *sync.Mutex
		store map[string]*Item
		queue Queue
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*Item
		wantErr bool
	}{
		{"test", fields{Mutex: &sync.Mutex{}, store: map[string]*Item{"a": {ID: "a"}}, queue: &testQueue{}}, args{ctx: context.Background()}, []*Item{{ID: "a"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &inMemQueueRepo{
				Mutex: tt.fields.Mutex,
				store: tt.fields.store,
				queue: tt.fields.queue,
			}
			got, err := r.List(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("queueRepository.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queueRepository.List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queueRepository_Concurrency(t *testing.T) {
	ctx := context.Background()
	q := NewQueueRepository(&testQueue{})
	wg := sync.WaitGroup{}
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		for _, jd := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
			wg.Add(3)
			test := fmt.Sprintf("%s%s", id, jd)
			go func(test string) {
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
			go func(test string) {
				_, err := q.Get(ctx, test)
				if err != nil && err != ErrNotFound {
					t.Error(err)
				}
				wg.Done()
			}(test)
		}
	}
	wg.Wait()
}

func Test_queueRepository_Get(t *testing.T) {
	type fields struct {
		Mutex *sync.Mutex
		store map[string]*Item
		queue Queue
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Item
		wantErr bool
	}{
		{"ok", fields{Mutex: &sync.Mutex{}, store: map[string]*Item{"test": {ID: "test"}}, queue: &testQueue{}}, args{ctx: context.Background(), id: "test"}, &Item{ID: "test"}, false},
		{"missing", fields{Mutex: &sync.Mutex{}, store: map[string]*Item{"hi": {ID: "no thanks"}}, queue: &testQueue{}}, args{ctx: context.Background(), id: "test"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &inMemQueueRepo{
				Mutex: tt.fields.Mutex,
				store: tt.fields.store,
				queue: tt.fields.queue,
			}
			got, err := r.Get(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("queueRepository.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queueRepository.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queueRepository_Create(t *testing.T) {
	type fields struct {
		Mutex *sync.Mutex
		store map[string]*Item
		queue Queue
	}
	type args struct {
		ctx context.Context
		req Item
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"ok", fields{Mutex: &sync.Mutex{}, store: map[string]*Item{}, queue: &testQueue{}}, args{ctx: context.Background(), req: Item{ID: "test"}}, false},
		{"baditem", fields{Mutex: &sync.Mutex{}, store: map[string]*Item{}, queue: &testQueue{}}, args{ctx: context.Background(), req: Item{ID: ""}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &inMemQueueRepo{
				Mutex: tt.fields.Mutex,
				store: tt.fields.store,
				queue: tt.fields.queue,
			}
			if err := r.Create(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("queueRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				_, ok := r.store[tt.args.req.ID]
				assert.Assert(t, ok)
			}
		})
	}
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
