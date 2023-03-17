// Dag provides a way to describe a directed acyclic graph of work to be done.
// It starts with a root node, then you add nodes to it. Outputs are
// automatically connected to subsequent inputs.
package dag

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type any interface{}

type NodeMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

type Node[T any] struct {
	*sync.RWMutex
	work     Work[T]
	children []*Node[T]
	input    T
	output   T
	meta     NodeMetadata
}

type Work[T any] func(context.Context, T) T

func NewNode[T any](job Work[T], input T) *Node[T] {
	return &Node[T]{
		RWMutex: &sync.RWMutex{},
		work:    job,
		input:   input,
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

// Output gets a thread safe copy of the output of the node
func (n *Node[T]) Output() T {
	n.RLock()
	defer n.RUnlock()
	return n.output
}

// Children returns all the node's children
func (n *Node[T]) Children() []*Node[T] {
	n.RLock()
	defer n.RUnlock()
	return n.children
}

// Meta returns the node's metadata
func (n *Node[T]) Meta() NodeMetadata {
	n.RLock()
	defer n.RUnlock()
	return n.meta
}

func (n *Node[T]) AddChild(job Work[T]) *Node[T] {
	n.Lock()
	defer n.Unlock()
	if n.children == nil {
		n.children = []*Node[T]{}
	}
	node := NewNode(job, n.output)
	n.children = append(n.children, node)
	return node
}

func (n *Node[T]) Execute(ctx context.Context) {
	if n == nil {
		log.Ctx(ctx).Error().Msg("dag incorrectly formed, ignoring")
		return
	}
	n.setStart()
	n.setOutput(n.work(ctx, n.input))
	n.setEnd()
	for _, child := range n.children {
		child.setInput(n.Output())
		child.Execute(ctx)
	}
}

func (n *Node[T]) setStart() {
	n.Lock()
	defer n.Unlock()
	n.meta.StartedAt = time.Now()
}

func (n *Node[T]) setEnd() {
	n.Lock()
	defer n.Unlock()
	n.meta.EndedAt = time.Now()
}

func (n *Node[T]) setOutput(output T) {
	n.Lock()
	defer n.Unlock()
	n.output = output
}

func (n *Node[T]) setInput(input T) {
	n.Lock()
	defer n.Unlock()
	n.input = input
}
