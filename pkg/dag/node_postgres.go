package dag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/util"
)

type postgresNode struct {
	mu          sync.RWMutex
	persistence db.NodePersistence
	workRepo    WorkRepository[IOSpec]
	id          int32
}

func NewPostgresNode(ctx context.Context, persistence db.NodePersistence, workRepo WorkRepository[IOSpec], n NodeSpec[IOSpec]) (Node[IOSpec], error) {
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
	return newPostgresNodeWithID(persistence, workRepo, id), nil
}

func newPostgresNodeWithID(persistence db.NodePersistence, workRepo WorkRepository[IOSpec], id int32) Node[IOSpec] {
	return &postgresNode{
		persistence: persistence,
		id:          id,
		workRepo:    workRepo,
	}
}

func (n *postgresNode) ID() int32 {
	return n.id
}

func (n *postgresNode) AddParentChildRelationship(ctx context.Context, child Node[IOSpec]) error {
	if err := n.AddChild(ctx, child); err != nil {
		return err
	}
	if err := child.AddParent(ctx, n); err != nil {
		return err
	}
	return nil
}

func (n *postgresNode) AddChild(ctx context.Context, child Node[IOSpec]) error {
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

func (n *postgresNode) AddParent(ctx context.Context, parent Node[IOSpec]) error {
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

func (n *postgresNode) AddInput(ctx context.Context, input IOSpec) error {
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

func (n *postgresNode) AddOutput(ctx context.Context, output IOSpec) error {
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

func (n *postgresNode) Get(ctx context.Context) (NodeRepresentation[IOSpec], error) {
	dbNode, err := n.persistence.GetNodeByID(ctx, n.id)
	if err != nil {
		return NodeRepresentation[IOSpec]{}, getPostgresNodeError(err)
	}
	var parents []Node[IOSpec]
	for _, parentId := range dbNode.Parents {
		parents = append(parents, newPostgresNodeWithID(n.persistence, n.workRepo, parentId))
	}
	var children []Node[IOSpec]
	for _, childId := range dbNode.Children {
		children = append(children, newPostgresNodeWithID(n.persistence, n.workRepo, childId))
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

func (n *postgresNode) SetMetadata(c context.Context, meta NodeMetadata) error {
	return n.persistence.CreateStatus(c, db.CreateStatusParams{
		NodeID:    n.id,
		Submitted: meta.CreatedAt,
		Started:   util.NullTime(meta.StartedAt),
		Ended:     util.NullTime(meta.EndedAt),
		Status:    meta.Status,
	})
}

func (n *postgresNode) SetResults(c context.Context, result NodeResult) error {
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
