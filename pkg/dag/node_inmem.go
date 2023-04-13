package dag

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

type inMemNode[T any] struct {
	// This is a tricky data structure to get right, because we need
	// to mutate elements in a thread-safe way, without deadlocking. The simplest
	// way to achieve this is with lots of small setters.
	mu          sync.RWMutex
	id          int32 // Keep track of the node's ID, useful during debugging
	name        string
	queueItemID uuid.UUID    // ID of the queue item
	children    []Node[T]    // Children of the node
	parents     []Node[T]    // Parents of the node
	inputs      []T          // Input data
	outputs     []T          // Output data (which is fed into the inputs of its children)
	meta        NodeMetadata // Metadata about the node
	result      NodeResult   // Result of the node
	workRepo    WorkRepository[T]
}

func NewInMemoryNode[T any](ctx context.Context, workRepo WorkRepository[T], n NodeSpec[T]) (Node[T], error) {
	id := rand.Int31()
	err := workRepo.Set(ctx, id, n.Work)
	if err != nil {
		return nil, err
	}
	node := newInMemoryNodeWithID(workRepo, n, id)
	return node, nil
}

func newInMemoryNodeWithID[T any](workRepo WorkRepository[T], n NodeSpec[T], id int32) Node[T] {
	return &inMemNode[T]{
		id:       id,
		name:     n.Name,
		workRepo: workRepo,
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

func (n *inMemNode[T]) ID() int32 {
	return n.id
}

func (n *inMemNode[T]) AddParentChildRelationship(ctx context.Context, child Node[T]) error {
	if err := n.AddChild(ctx, child); err != nil {
		return err
	}
	if err := child.AddParent(ctx, n); err != nil {
		return err
	}
	return nil
}

func (n *inMemNode[T]) AddChild(ctx context.Context, child Node[T]) error {
	n.mu.Lock()
	n.children = append(n.children, child)
	n.mu.Unlock()
	return nil
}

func (n *inMemNode[T]) AddParent(ctx context.Context, parent Node[T]) error {
	n.mu.Lock()
	n.parents = append(n.parents, parent)
	n.mu.Unlock()
	return nil
}

func (n *inMemNode[T]) AddInput(ctx context.Context, input T) error {
	n.inputs = append(n.inputs, input)
	return nil
}

func (n *inMemNode[T]) AddOutput(ctx context.Context, output T) error {
	n.outputs = append(n.outputs, output)
	return nil
}

func (n *inMemNode[T]) Get(ctx context.Context) (NodeRepresentation[T], error) {
	w, err := n.workRepo.Get(ctx, n.id)
	if err != nil {
		return NodeRepresentation[T]{}, err
	}
	return NodeRepresentation[T]{
		Id:          n.id,
		Name:        n.name,
		QueueItemID: n.queueItemID,
		Work:        w,
		Children:    n.children,
		Parents:     n.parents,
		Inputs:      n.inputs,
		Outputs:     n.outputs,
		Metadata:    n.meta,
		Results:     n.result,
	}, nil
}

func (n *inMemNode[T]) SetMetadata(c context.Context, meta NodeMetadata) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.meta = meta
	return nil
}

func (n *inMemNode[T]) SetResults(c context.Context, result NodeResult) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.result = result
	return nil
}
