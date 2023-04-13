package dag

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestLinearDag(t *testing.T) {
	ctx := context.Background()
	wr := NewInMemWorkRepository[int]()
	root, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, input []int, c chan NodeResult) []int {
			defer close(c)
			return []int{1}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	child1, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			return []int{inputs[0] + 1}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child1)
	assert.NilError(t, err)
	child2, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			return []int{inputs[0] + 1}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = child1.AddParentChildRelationship(ctx, child2)
	assert.NilError(t, err)
	Execute(ctx, root)
	r, err := root.Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, r.Outputs[0], 1)
	c1, err := r.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, c1.Outputs[0], 2)
	c2, err := c1.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, c2.Outputs[0], 3)
}

func TestForkingDag(t *testing.T) {
	ctx := context.Background()
	wr := NewInMemWorkRepository[int]()
	root, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			return []int{2}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	child1, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			return []int{inputs[0] + 1}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child1)
	assert.NilError(t, err)
	child2, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			return []int{inputs[0] * 2}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child2)
	assert.NilError(t, err)
	Execute(ctx, root)
	r, err := root.Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, r.Outputs[0], 2)
	c1, err := r.Children[0].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, c1.Outputs[0], 3)
	c2, err := r.Children[1].Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, c2.Outputs[0], 4)
}

func TestMapReduceDag(t *testing.T) {
	ctx := context.Background()
	wr := NewInMemWorkRepository[int]()
	root, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			assert.Equal(t, len(inputs), 1)
			return []int{inputs[0]}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddInput(ctx, 1)
	assert.NilError(t, err)
	child1, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			assert.Equal(t, len(inputs), 1)
			return []int{inputs[0] * 3}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child1)
	assert.NilError(t, err)
	child1a, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			assert.Equal(t, len(inputs), 1)
			return []int{inputs[0] * 3}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = child1.AddParentChildRelationship(ctx, child1a)
	assert.NilError(t, err)
	child2, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			assert.Equal(t, len(inputs), 1)
			return []int{inputs[0] * 2}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child2)
	assert.NilError(t, err)
	child3, err := NewInMemoryNode(ctx, wr, NodeSpec[int]{
		Work: func(ctx context.Context, inputs []int, c chan NodeResult) []int {
			defer close(c)
			assert.Equal(t, len(inputs), 2)
			var i int
			for _, v := range inputs {
				i += v
			}
			return []int{i}
		},
		OwnerID: uuid.New(),
	})
	assert.NilError(t, err)
	err = child1a.AddParentChildRelationship(ctx, child3)
	assert.NilError(t, err)
	err = child2.AddParentChildRelationship(ctx, child3)
	assert.NilError(t, err)
	Execute(ctx, root)
	// Remember that all inputs are added to the next output. So brach 1 will
	// Have three outputs, root, child1, and child1a.
	c3, err := child3.Get(ctx)
	assert.NilError(t, err)
	assert.Equal(t, c3.Outputs[0], 11)
}

func TestTimeIsMonotonic(t *testing.T) {
	ctx := context.Background()
	wr := NewInMemWorkRepository[interface{}]()
	ns := NodeSpec[interface{}]{
		Work: func(ctx context.Context, inputs []interface{}, c chan NodeResult) []interface{} {
			defer close(c)
			time.Sleep(100 * time.Microsecond)
			return []interface{}{nil}
		},
		OwnerID: uuid.New(),
	}
	root, err := NewInMemoryNode(ctx, wr, ns)
	assert.NilError(t, err)
	child1, err := NewInMemoryNode(ctx, wr, ns)
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child1)
	assert.NilError(t, err)
	child2, err := NewInMemoryNode(ctx, wr, ns)
	assert.NilError(t, err)
	err = child1.AddParentChildRelationship(ctx, child2)
	assert.NilError(t, err)
	Execute(ctx, root)
	r, err := root.Get(ctx)
	assert.NilError(t, err)
	assert.Assert(t, r.Metadata.StartedAt.Before(r.Metadata.EndedAt))
	c1, err := child1.Get(ctx)
	assert.NilError(t, err)
	c2, err := child2.Get(ctx)
	assert.NilError(t, err)
	assert.Assert(t, c1.Metadata.EndedAt.Before(c2.Metadata.EndedAt))
}

// This kind of structure is used in the queue, naughty, naughty.
// Test that it doesn't panic. Uber defensive.
func TestNilDag(t *testing.T) {
	var d Node[string]
	Execute(context.Background(), d)
}

// Found an issue where the status (a chan) wasn't responding quick enough
func TestStatusDoesntRace(t *testing.T) {
	ctx := context.Background()
	wr := NewInMemWorkRepository[interface{}]()
	rootNodeSpec := NodeSpec[interface{}]{
		Work: func(ctx context.Context, input []interface{}, c chan NodeResult) []interface{} {
			defer close(c)
			c <- NodeResult{Skipped: true}
			return []interface{}{nil}
		},
		OwnerID: uuid.New(),
	}
	root, err := NewInMemoryNode(ctx, wr, rootNodeSpec)
	assert.NilError(t, err)
	childNodeSpec := NodeSpec[interface{}]{
		Work: func(ctx context.Context, input []interface{}, c chan NodeResult) []interface{} {
			go func() {
				time.Sleep(100 * time.Millisecond)
				c <- NodeResult{Skipped: true}
				close(c)
			}()
			return []interface{}{nil}
		},
		OwnerID: uuid.New(),
	}
	child, err := NewInMemoryNode(ctx, wr, childNodeSpec)
	assert.NilError(t, err)
	err = root.AddParentChildRelationship(ctx, child)
	assert.NilError(t, err)
	Execute(ctx, root)
	c1, err := child.Get(ctx)
	assert.NilError(t, err)
	assert.Assert(t, c1.Results.Skipped)
}
