package dag

import (
	"context"
	"fmt"
	"sync"
)

// Work is shorthand for a function that accepts inputs and returns outputs.
type Work[T any] func(ctx context.Context, inputs []T, statusChan chan NodeResult) []T

var (
	ErrWorkNotFound      = fmt.Errorf("work not found")
	ErrWorkAlreadyExists = fmt.Errorf("work already exists")
)

type WorkRepository[T any] interface {
	Get(context.Context, int32) (Work[T], error)
	Set(context.Context, int32, Work[T]) error
}

type inMemWorkRepository[T any] struct {
	mu   sync.RWMutex
	work map[int32]Work[T]
}

func NewInMemWorkRepository[T any]() WorkRepository[T] {
	return &inMemWorkRepository[T]{
		work: make(map[int32]Work[T]),
	}
}

func (r *inMemWorkRepository[T]) Get(ctx context.Context, k int32) (Work[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.work[k]
	if !ok {
		return nil, ErrWorkNotFound
	}
	return w, nil
}

func (r *inMemWorkRepository[T]) Set(ctx context.Context, k int32, w Work[T]) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.work[k]
	if ok {
		return ErrWorkAlreadyExists
	}
	r.work[k] = w
	return nil
}

func NilFunc(ctx context.Context, inputs []IOSpec, statusChan chan NodeResult) []IOSpec {
	defer close(statusChan)
	statusChan <- NodeResult{
		StdOut: "ok",
	}
	return nil
}
