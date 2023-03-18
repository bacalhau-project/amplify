package dag

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestLinearDag(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, inputs []int) []int {
		return []int{1}
	}, []int{0})
	child1 := NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] + 1}
	})
	root.AddChild(child1)
	child2 := NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] + 1}
	})
	child1.AddChild(child2)
	root.Execute(ctx)
	assert.Equal(t, root.Outputs()[0], 1)
	assert.Equal(t, root.Children()[0].Outputs()[0], 2)
	assert.Equal(t, root.Children()[0].Children()[0].Outputs()[0], 3)
}

func TestForkingDag(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, inputs []int) []int {
		return []int{2}
	}, []int{0})
	child1 := NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] + 1}
	})
	root.AddChild(child1)
	child2 := NewNode(func(ctx context.Context, inputs []int) []int {
		return []int{inputs[0] * 2}
	})
	root.AddChild(child2)
	root.Execute(ctx)
	assert.Equal(t, root.Outputs()[0], 2)
	assert.Equal(t, child1.Outputs()[0], 3)
	assert.Equal(t, child2.Outputs()[0], 4)
}

func TestMapReduceDag(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, inputs []int) []int {
		assert.Equal(t, len(inputs), 1)
		return []int{inputs[0]}
	}, []int{1})
	child1 := NewNode(func(ctx context.Context, inputs []int) []int {
		assert.Equal(t, len(inputs), 1)
		return []int{inputs[0] * 3}
	})
	root.AddChild(child1)
	child1a := NewNode(func(ctx context.Context, inputs []int) []int {
		assert.Equal(t, len(inputs), 1)
		return []int{inputs[0] * 3}
	})
	child1.AddChild(child1a)
	child2 := NewNode(func(ctx context.Context, inputs []int) []int {
		assert.Equal(t, len(inputs), 1)
		return []int{inputs[0] * 2}
	})
	root.AddChild(child2)
	child3 := NewNode(func(ctx context.Context, inputs []int) []int {
		assert.Equal(t, len(inputs), 2)
		var i int
		for _, v := range inputs {
			i += v
		}
		return []int{i}
	})
	child1a.AddChild(child3)
	child2.AddChild(child3)
	root.Execute(ctx)
	// Remember that all inputs are added to the next output. So brach 1 will
	// Have three outputs, root, child1, and child1a.
	assert.Equal(t, child3.Outputs()[0], 11)
}

func TestTimeIsMonotonic(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, input []interface{}) []interface{} {
		time.Sleep(100 * time.Microsecond)
		return []interface{}{nil}
	}, nil)
	child1 := NewNode(func(ctx context.Context, input []interface{}) []interface{} {
		time.Sleep(100 * time.Microsecond)
		return []interface{}{nil}
	})
	root.AddChild(child1)
	child2 := NewNode(func(ctx context.Context, input []interface{}) []interface{} {
		time.Sleep(100 * time.Microsecond)
		return []interface{}{nil}
	})
	child1.AddChild(child2)
	root.Execute(ctx)
	assert.Assert(t, root.Meta().StartedAt.Before(root.Meta().EndedAt))
	assert.Assert(t, child1.Meta().EndedAt.Before(child2.Meta().EndedAt))
}

// This kind of structure is used in the queue, naughty, naughty.
// Test that it doesn't panic. Uber defensive.
func TestNilDag(t *testing.T) {
	var d *Node[[]string]
	d.Execute(context.Background())
}
