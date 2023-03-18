package dag

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestLinearDag(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, input int) int {
		return 1
	}, 0)
	child := root.AddChild(func(ctx context.Context, input int) int {
		return input + 1
	})
	child.AddChild(func(ctx context.Context, input int) int {
		return input + 1
	})
	root.Execute(ctx)
	assert.Equal(t, root.Output(), 1)
	assert.Equal(t, root.Children()[0].Output(), 2)
	assert.Equal(t, root.Children()[0].Children()[0].Output(), 3)
}

func TestForkingDag(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, input int) int {
		return 2
	}, 0)
	child1 := root.AddChild(func(ctx context.Context, input int) int {
		return input + 1
	})
	child2 := root.AddChild(func(ctx context.Context, input int) int {
		return input * 2
	})
	root.Execute(ctx)
	assert.Equal(t, root.Output(), 2)
	assert.Equal(t, child1.Output(), 3)
	assert.Equal(t, child2.Output(), 4)
}

func TestTimeIsMonotonic(t *testing.T) {
	ctx := context.Background()
	root := NewDag(func(ctx context.Context, input interface{}) interface{} {
		time.Sleep(1 * time.Microsecond)
		return 1
	}, nil)
	child := root.AddChild(func(ctx context.Context, input interface{}) interface{} {
		time.Sleep(1 * time.Microsecond)
		return 2
	})
	child.AddChild(func(ctx context.Context, input interface{}) interface{} {
		time.Sleep(1 * time.Microsecond)
		return 3
	})
	root.Execute(ctx)
	assert.Assert(t, root.Meta().StartedAt.Before(root.Meta().EndedAt))
	assert.Assert(t, root.Children()[0].Meta().StartedAt.Before(root.Children()[0].Meta().EndedAt))
	assert.Assert(t, root.Children()[0].Children()[0].Meta().StartedAt.Before(root.Children()[0].Children()[0].Meta().EndedAt))
}

// This kind of structure is used in the queue, naughty, naughty.
// Test that it doesn't panic. Uber defensive.
func TestNilDag(t *testing.T) {
	var d *Node[[]string]
	d.execute(context.Background(), nil)
}
