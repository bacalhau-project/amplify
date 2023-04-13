package dag

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type NodeRepresentation[T any] struct {
	Id          int32        // Keep track of the node's ID, useful during debugging
	Name        string       // Name of the node
	QueueItemID uuid.UUID    // ID of the queue item
	Work        Work[T]      // The work to be done by the node
	Children    []Node[T]    // Children of the node
	Parents     []Node[T]    // Parents of the node
	Inputs      []T          // Input data
	Outputs     []T          // Output data (which is fed into the inputs of its children)
	Metadata    NodeMetadata // Metadata about the node
	Results     NodeResult   // Result of the node
}

// NodeMetadata contains metadata about a node
type NodeMetadata struct {
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
	Status    string // Status of the execution
}

type NodeResult struct {
	ID      string // External ID of the execution
	StdOut  string // Stdout of the execution
	StdErr  string // Stderr of the execution
	Skipped bool   // Whether the execution was skipped
}

type Status int64

const (
	Queued Status = iota
	Started
	Finished
)

func (s Status) String() string {
	switch s {
	case Queued:
		return "queued"
	case Started:
		return "started"
	case Finished:
		return "finished"
	}
	return "unknown"
}

type Node[T any] interface {
	ID() int32
	Get(context.Context) (NodeRepresentation[T], error)
	AddChild(context.Context, Node[T]) error
	AddParentChildRelationship(context.Context, Node[T]) error
	AddParent(context.Context, Node[T]) error
	AddInput(context.Context, T) error
	AddOutput(context.Context, T) error
	SetMetadata(context.Context, NodeMetadata) error
	SetResults(context.Context, NodeResult) error
}

type NodeSpec[T any] struct {
	OwnerID uuid.UUID
	Name    string
	Work    Work[T]
}

// Execute a given node
func Execute[T any](ctx context.Context, node Node[T]) {
	if node == nil {
		log.Ctx(ctx).Error().Msg("node is nil")
		return
	}
	// Get a copy of the node representation
	n, err := node.Get(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error getting node")
		return
	}

	// Check if all the parents are ready
	ready := true
	for _, parent := range n.Parents {
		p, err := parent.Get(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error getting parent")
			return
		}
		if p.Metadata.EndedAt.IsZero() {
			ready = false
			break
		}
	}
	if !ready {
		log.Ctx(ctx).Debug().Int32("id", n.Id).Msg("parent not ready, waiting")
		return
	}
	// Check if this node is/has already been executed
	if !n.Metadata.StartedAt.IsZero() {
		log.Ctx(ctx).Debug().Int32("id", n.Id).Msg("already started, skipping")
		return
	}
	n, err = updateNodeStartTime(ctx, node) // Set the start time
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error updating node start time")
		return
	}
	resultChan := make(chan NodeResult, 10)      // Channel to receive status updates, must close once complete
	outputs := n.Work(ctx, n.Inputs, resultChan) // Do the work
	for status := range resultChan {             // Block waiting for the status
		err = node.SetResults(ctx, status)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error setting node results")
			return
		}
	}
	for _, output := range outputs { // Add the outputs to the node
		err = node.AddOutput(ctx, output)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error adding output")
			return
		}
	}
	// Append results to the inputs of all children
	for _, child := range n.Children {
		// Append the outputs of this node to the inputs of the child. The
		// actual execution decides whether to use them.
		for _, output := range outputs { // Add the outputs to the node
			err = child.AddInput(ctx, output)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error adding inputs to child")
				return
			}
		}
	}
	n, err = updateNodeEndTime(ctx, node) // Set the end time
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Int32("id", n.Id).Msg("error updating node end time")
		return
	}

	// Execute all children
	for _, child := range n.Children {
		Execute(ctx, child)
	}
}

func updateNodeStartTime[T any](ctx context.Context, node Node[T]) (NodeRepresentation[T], error) {
	n, err := node.Get(ctx)
	if err != nil {
		return n, err
	}
	err = node.SetMetadata(ctx, NodeMetadata{
		CreatedAt: n.Metadata.CreatedAt,
		StartedAt: time.Now(),
		Status:    Started.String(),
	})
	if err != nil {
		return n, err
	}
	return node.Get(ctx)
}

func updateNodeEndTime[T any](ctx context.Context, node Node[T]) (NodeRepresentation[T], error) {
	n, err := node.Get(ctx)
	if err != nil {
		return n, err
	}
	err = node.SetMetadata(ctx, NodeMetadata{
		CreatedAt: n.Metadata.CreatedAt,
		StartedAt: n.Metadata.StartedAt,
		EndedAt:   time.Now(),
		Status:    Finished.String(),
	})
	if err != nil {
		return n, err
	}
	return node.Get(ctx)
}
