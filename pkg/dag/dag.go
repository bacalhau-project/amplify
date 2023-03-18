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

type any comparable

// NodeMetadata contains metadata about a node
type NodeMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

// Work is shorthand for a function that accepts inputs and returns outputs.
type Work[T any] func(ctx context.Context, inputs []T, statusChan chan NodeStatus) []T

// Node is a node in a directed acyclic graph. It has edges via links to
// child nodes.
type Node[T any] struct {
	// This is a tricky data structure to get right, because we need
	// to mutate elements in a thread-safe way, without deadlocking. The simplest
	// way to achieve this is with lots of small setters.
	mu       sync.RWMutex
	id       string       // Keep track of the node's ID, useful during debugging
	work     Work[T]      // The work to be done by the node
	children []*Node[T]   // Children of the node
	parents  []*Node[T]   // Parents of the node
	inputs   []T          // Input data
	outputs  []T          // Output data (which is fed into the inputs of its children)
	meta     NodeMetadata // Metadata about the node
	status   NodeStatus   // Status of the node
}

type NodeStatus struct {
	ID     string // External ID of the execution
	StdOut string // Stdout of the execution
	StdErr string // Stderr of the execution
	Status string // Status of the execution
}

// NewDag creates a new dag with the given work and initial input
func NewDag[T any](id string, job Work[T], rootInputs []T) *Node[T] {
	return &Node[T]{
		id:       id,
		work:     job,
		inputs:   rootInputs,
		children: []*Node[T]{},
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

func NewNode[T any](id string, job Work[T]) *Node[T] {
	return &Node[T]{
		id:       id,
		work:     job,
		children: []*Node[T]{},
		meta: NodeMetadata{
			CreatedAt: time.Now(),
		},
	}
}

// AddChild creates a child Node from some work
func (n *Node[T]) AddChild(node *Node[T]) {
	node.mu.Lock()
	node.parents = append(node.parents, n)
	node.mu.Unlock()
	n.mu.Lock()
	n.children = append(n.children, node)
	n.mu.Unlock()
}

// Internal method to execute the node and all its children given an input
func (n *Node[T]) Execute(ctx context.Context) {
	// Be careful with deadlocking here, because we're mutating itself and
	// its children. Can't lock the node while we're executing its children.
	if n == nil {
		log.Ctx(ctx).Error().Msg("dag incorrectly formed, ignoring")
		return
	}
	// Check if all the parents are ready
	ready := true
	for _, parent := range n.Parents() {
		if !parent.Complete() {
			ready = false
			break
		}
	}
	if !ready {
		log.Ctx(ctx).Debug().Str("id", n.id).Msg("parent not ready, skipping")
		return
	}
	// Check if this node is/has already been executed
	if n.Started() {
		log.Ctx(ctx).Debug().Str("id", n.id).Msg("already complete, skipping")
		return
	}
	n.mu.Lock()                         // Lock the node
	n.meta.StartedAt = time.Now()       // Set the start time
	n.mu.Unlock()                       // Unlock the node for execution
	statusChan := make(chan NodeStatus) // Channel to receive status updates
	statusCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func(node *Node[T]) { // Goroutine to receive status updates
		for {
			select {
			case <-statusCtx.Done():
				return
			case status := <-statusChan:
				node.mu.Lock()
				node.status = status
				node.mu.Unlock()
			}
		}
	}(n)
	outputs := n.work(ctx, n.Inputs(), statusChan) // Do the work
	n.mu.Lock()                                    // Lock the node
	n.outputs = outputs                            // Record the output
	n.meta.EndedAt = time.Now()                    // Set the end time
	n.mu.Unlock()                                  // Unlock the node for children
	cancel()                                       // Cancel the status context

	// Append results to the inputs of all children
	for _, child := range n.Children() {
		// Append the outputs of this node to the inputs of the child. The
		// actual execution decides whether to use them.
		child.mu.Lock()
		child.inputs = append(child.inputs, n.Outputs()...)
		child.mu.Unlock()
	}

	// Execute all children
	for _, child := range n.Children() {
		child.Execute(ctx)
	}
}

// Input gets the inputs of a node
func (n *Node[T]) Inputs() []T {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.inputs
}

// Output gets the outputs of a node
func (n *Node[T]) Outputs() []T {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.outputs
}

// Children returns all the node's children
func (n *Node[T]) Children() []*Node[T] {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.children
}

// Parents returns all the node's parents
func (n *Node[T]) Parents() []*Node[T] {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.parents
}

// Meta returns the node's metadata
func (n *Node[T]) Meta() NodeMetadata {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.meta
}

func (n *Node[T]) Started() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return !n.meta.StartedAt.IsZero()
}

func (n *Node[T]) Complete() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return !n.meta.EndedAt.IsZero()
}

func (n *Node[T]) Status() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}
