// Dag provides a way to describe a directed acyclic graph of work to be done.
// It starts with a root node, then you add nodes to it. Outputs are
// automatically connected to subsequent inputs.
package dag

import (
	"context"
	"sync"
	"time"
)

type any interface{}

type Node[T any] struct {
	*sync.Mutex
	Work     Work[T]
	Children []*Node[T]
	Input    T
	Output   T
	Created  time.Time
	Started  time.Time
	Ended    time.Time
}

type Work[T any] func(context.Context, T) T

func NewNode[T any](job Work[T], input T) *Node[T] {
	return &Node[T]{
		Mutex:   &sync.Mutex{},
		Work:    job,
		Input:   input,
		Created: time.Now(),
	}
}

func (n *Node[T]) AddChild(job Work[T]) *Node[T] {
	n.Lock()
	defer n.Unlock()
	if n.Children == nil {
		n.Children = []*Node[T]{}
	}
	node := NewNode(job, n.Output)
	n.Children = append(n.Children, node)
	return node
}

func (n *Node[T]) Execute(ctx context.Context) {
	n.Lock()
	n.Started = time.Now()
	n.Output = n.Work(ctx, n.Input)
	n.Ended = time.Now()
	n.Unlock()
	for _, child := range n.Children {
		child.Input = n.Output
		child.Execute(ctx)
	}
}
