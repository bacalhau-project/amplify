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

// NodeMetadata contains metadata about a node
type NodeMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

// Work is shorthand for a function that accepts inputs and returns outputs.
type Work[T any] func(ctx context.Context, input T) T

// Node is a node in a directed acyclic graph. It has edges via links to
// child nodes.
type Node[T any] struct {
	// This is a tricky data structure to get right, because we need
	// to mutate elements in a thread-safe way, without deadlocking. The simplest
	// way to achieve this is with lots of small setters.
	mu       sync.RWMutex
	work     Work[T]      // The work to be done by the node
	children []*Node[T]   // Children of the node
	input    T            // Input data
	output   T            // Output data (which is fed into the inputs of its children)
	meta     NodeMetadata // Metadata about the node
}

// NewDag creates a new dag with the given work and initial input
func NewDag[T any](job Work[T], rootInput T) *Node[T] {
	return &Node[T]{
		work:     job,
		input:    rootInput,
		children: []*Node[T]{},
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

func newNode[T any](job Work[T]) *Node[T] {
	return &Node[T]{
		work:     job,
		children: []*Node[T]{},
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

// AddChild creates a child Node from some work
func (n *Node[T]) AddChild(job Work[T]) *Node[T] {
	node := newNode(job)
	n.mu.Lock()
	n.children = append(n.children, node)
	n.mu.Unlock()
	return node
}

// Execute runs the work of the node and all it's children
func (n *Node[T]) Execute(ctx context.Context) {
	n.execute(ctx, n.Input())
}

// Internal method to execute the node and all its children given an input
func (n *Node[T]) execute(ctx context.Context, input T) {
	// Be careful with deadlocking here, because we're mutating itself and
	// its children. Can't lock the node while we're executing its children.
	if n == nil {
		log.Ctx(ctx).Error().Msg("dag incorrectly formed, ignoring")
		return
	}
	n.mu.Lock()                        // Lock the node
	n.meta.StartedAt = time.Now()      // Set the start time
	n.input = input                    // Record the input
	output := n.work(ctx, input)       // Do the work
	n.output = output                  // Record the output
	n.meta.EndedAt = time.Now()        // Set the end time
	n.mu.Unlock()                      // Unlock the node for children
	for _, child := range n.children { // Execute all children
		child.execute(ctx, output) // Input is the output of the parent
	}
}

// Input gets the inputs of a node
func (n *Node[T]) Input() T {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.input
}

// Output gets the outputs of a node
func (n *Node[T]) Output() T {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.output
}

// Children returns all the node's children
func (n *Node[T]) Children() []*Node[T] {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.children
}

// Meta returns the node's metadata
func (n *Node[T]) Meta() NodeMetadata {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.meta
}
