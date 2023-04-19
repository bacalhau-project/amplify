package dag

import (
	"context"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

// Should create a new node in the database, add the work to the work repository
// and return a node
func TestNewPostgresNode(t *testing.T) {
	ctx := context.Background()
	p := &mockNodePersistence{}
	wr := NewInMemWorkRepository[IOSpec]()
	n := NodeSpec[IOSpec]{
		OwnerID: uuid.New(),
		Name:    "test",
		Work:    testWork,
	}
	_, err := NewNode(ctx, p, wr, n)
	assert.NilError(t, err)
	assert.Equal(t, p.NodeID, int32(1))
	_, err = wr.Get(ctx, 0)
	assert.NilError(t, err)
}

func testWork(ctx context.Context, input []IOSpec, c chan NodeResult) []IOSpec {
	defer close(c)
	c <- NodeResult{Skipped: true}
	return []IOSpec{}
}

// Should add links to parent and child nodes
func Test_postgresNode_AddParentChildRelationship(t *testing.T) {
	ctx := context.Background()
	p := &mockNodePersistence{}
	wr := NewInMemWorkRepository[IOSpec]()
	n := NodeSpec[IOSpec]{
		OwnerID: uuid.New(),
		Name:    "test",
		Work:    testWork,
	}
	root, err := NewNode(ctx, p, wr, n)
	assert.NilError(t, err)
	child, err := NewNode(ctx, p, wr, n)
	assert.NilError(t, err)
	err = root.AddChild(ctx, child)
	assert.NilError(t, err)
	assert.Equal(t, len(p.Links), 1)
	l, ok := p.Links[0]
	assert.Assert(t, ok)
	assert.Equal(t, l, int32(1))
}

var _ db.NodePersistence = (*mockNodePersistence)(nil)

type mockNodePersistence struct {
	NodeID int32
	Links  map[int32]int32
}

func (m *mockNodePersistence) CreateEdge(ctx context.Context, arg db.CreateEdgeParams) error {
	if m.Links == nil {
		m.Links = make(map[int32]int32)
	}
	m.Links[arg.ParentID] = arg.ChildID
	return nil
}

func (*mockNodePersistence) CreateIOSpec(ctx context.Context, arg db.CreateIOSpecParams) error {
	panic("unimplemented")
}

func (m *mockNodePersistence) CreateAndReturnNode(ctx context.Context, arg db.CreateAndReturnNodeParams) (db.Node, error) {
	n := m.NodeID
	m.NodeID++
	return db.Node{ID: n}, nil
}

func (*mockNodePersistence) GetIOSpecByID(ctx context.Context, id int32) (db.IoSpec, error) {
	panic("unimplemented")
}

func (*mockNodePersistence) GetNodeByID(ctx context.Context, id int32) (db.GetNodeByIDRow, error) {
	return db.GetNodeByIDRow{
		ID: util.NullInt32(id),
	}, nil
}

func (*mockNodePersistence) CreateStatus(ctx context.Context, arg db.CreateStatusParams) error {
	return nil
}

func (*mockNodePersistence) CreateResult(ctx context.Context, arg db.CreateResultParams) error {
	return nil
}
