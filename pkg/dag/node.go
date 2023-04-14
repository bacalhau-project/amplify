package dag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/util"
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

type node struct {
	mu          sync.RWMutex
	persistence db.NodePersistence
	workRepo    WorkRepository[IOSpec]
	id          int32
}

func NewNode(ctx context.Context, persistence db.NodePersistence, workRepo WorkRepository[IOSpec], n NodeSpec[IOSpec]) (Node[IOSpec], error) {
	id, err := persistence.CreateNodeReturnId(ctx, db.CreateNodeReturnIdParams{
		QueueItemID: n.OwnerID,
		Name:        n.Name,
	})
	if err != nil {
		return nil, newPostgresNodeError(err)
	}
	err = persistence.CreateStatus(ctx, db.CreateStatusParams{
		NodeID:    id,
		Submitted: time.Now(),
		Status:    Queued.String(),
	})
	if err != nil {
		return nil, newPostgresNodeError(err)
	}
	err = workRepo.Set(ctx, id, n.Work)
	if err != nil {
		return nil, newPostgresNodeError(err)
	}
	return nodeNodeWithID(persistence, workRepo, id), nil
}

func nodeNodeWithID(persistence db.NodePersistence, workRepo WorkRepository[IOSpec], id int32) Node[IOSpec] {
	return &node{
		persistence: persistence,
		id:          id,
		workRepo:    workRepo,
	}
}

func (n *node) ID() int32 {
	return n.id
}

func (n *node) AddParentChildRelationship(ctx context.Context, child Node[IOSpec]) error {
	if err := n.AddChild(ctx, child); err != nil {
		return err
	}
	if err := child.AddParent(ctx, n); err != nil {
		return err
	}
	return nil
}

func (n *node) AddChild(ctx context.Context, child Node[IOSpec]) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	c, err := child.Get(ctx)
	if err != nil {
		return err
	}
	err = n.persistence.CreateEdge(ctx, db.CreateEdgeParams{
		ParentID: n.id,
		ChildID:  c.Id,
	})
	if err != nil {
		return err
	}
	return nil
}

func (n *node) AddParent(ctx context.Context, parent Node[IOSpec]) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	p, err := parent.Get(ctx)
	if err != nil {
		return err
	}
	err = n.persistence.CreateEdge(ctx, db.CreateEdgeParams{
		ParentID: p.Id,
		ChildID:  n.id,
	})
	if err != nil {
		return err
	}
	return nil
}

func (n *node) AddInput(ctx context.Context, input IOSpec) error {
	err := n.persistence.CreateIOSpec(ctx, db.CreateIOSpecParams{
		NodeID:   n.id,
		Type:     "input",
		NodeName: input.NodeName(),
		InputID:  input.ID(),
		Root:     input.IsRoot(),
		Value:    util.NullStr(input.CID()),
		Path:     util.NullStr(input.Path()),
		Context:  util.NullStr(input.Context()),
	})
	if err != nil {
		return err
	}
	return nil
}

func (n *node) AddOutput(ctx context.Context, output IOSpec) error {
	return n.persistence.CreateIOSpec(ctx, db.CreateIOSpecParams{
		NodeID:   n.id,
		Type:     "output",
		NodeName: output.NodeName(),
		InputID:  output.ID(),
		Root:     output.IsRoot(),
		Value:    util.NullStr(output.CID()),
		Path:     util.NullStr(output.Path()),
		Context:  util.NullStr(output.Context()),
	})
}

func (n *node) Get(ctx context.Context) (NodeRepresentation[IOSpec], error) {
	dbNode, err := n.persistence.GetNodeByID(ctx, n.id)
	if err != nil {
		return NodeRepresentation[IOSpec]{}, getPostgresNodeError(err)
	}
	var parents []Node[IOSpec]
	for _, parentId := range dbNode.Parents {
		parents = append(parents, nodeNodeWithID(n.persistence, n.workRepo, parentId))
	}
	var children []Node[IOSpec]
	for _, childId := range dbNode.Children {
		children = append(children, nodeNodeWithID(n.persistence, n.workRepo, childId))
	}
	var inputs []IOSpec
	for _, inputId := range dbNode.Inputs {
		i, err := n.persistence.GetIOSpecByID(ctx, inputId)
		if err != nil {
			return NodeRepresentation[IOSpec]{}, getPostgresNodeError(err)
		}
		inputs = append(inputs, deserializeDBIOSpec(i))
	}
	var outputs []IOSpec
	for _, outputId := range dbNode.Outputs {
		i, err := n.persistence.GetIOSpecByID(ctx, outputId)
		if err != nil {
			return NodeRepresentation[IOSpec]{}, getPostgresNodeError(err)
		}
		outputs = append(outputs, deserializeDBIOSpec(i))
	}
	w, err := n.workRepo.Get(ctx, dbNode.ID.Int32)
	if err != nil && err != ErrWorkNotFound {
		return NodeRepresentation[IOSpec]{}, getPostgresNodeError(err)
	}
	node := NodeRepresentation[IOSpec]{
		Id:          dbNode.ID.Int32,
		Name:        dbNode.Name.String,
		QueueItemID: dbNode.QueueItemID.UUID,
		Work:        w,
		Results: NodeResult{
			ID:      dbNode.ExecutionID.String,
			StdOut:  dbNode.Stdout.String,
			StdErr:  dbNode.Stderr.String,
			Skipped: dbNode.Skipped.Bool,
		},
		Metadata: NodeMetadata{
			CreatedAt: dbNode.Submitted,
			StartedAt: dbNode.Started.Time,
			EndedAt:   dbNode.Ended.Time,
			Status:    dbNode.Status,
		},
		Parents:  parents,
		Children: children,
		Inputs:   inputs,
		Outputs:  outputs,
	}
	return node, nil
}

func (n *node) SetMetadata(c context.Context, meta NodeMetadata) error {
	return n.persistence.CreateStatus(c, db.CreateStatusParams{
		NodeID:    n.id,
		Submitted: meta.CreatedAt,
		Started:   util.NullTime(meta.StartedAt),
		Ended:     util.NullTime(meta.EndedAt),
		Status:    meta.Status,
	})
}

func (n *node) SetResults(c context.Context, result NodeResult) error {
	return n.persistence.CreateResult(c, db.CreateResultParams{
		NodeID:      n.id,
		ExecutionID: util.NullStr(result.ID),
		Stdout:      util.NullStr(result.StdOut),
		Stderr:      util.NullStr(result.StdErr),
		Skipped:     util.NullBool(result.Skipped),
	})
}

func getPostgresNodeError(err error) error {
	return fmt.Errorf("getting node from postgres: %w", err)
}

func newPostgresNodeError(err error) error {
	return fmt.Errorf("creating node in postgres: %w", err)
}

func deserializeDBIOSpec(i db.IoSpec) IOSpec {
	return ioSpec{
		nodeName: i.NodeName,
		id:       i.InputID,
		root:     i.Root,
		value:    i.Value.String,
		path:     i.Path.String,
		context:  i.Context.String,
	}
}
