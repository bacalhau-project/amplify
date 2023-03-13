package dag

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestDag(t *testing.T) {
	ctx := context.Background()
	root := NewNode(func(ctx context.Context, input int) int {
		return 1
	}, 0)
	root.AddChild(func(ctx context.Context, input int) int {
		return input + 1
	}).AddChild(func(ctx context.Context, input int) int {
		return input + 1
	})
	root.Execute(ctx)
	assert.Equal(t, root.Output, 1)
	assert.Equal(t, root.Children[0].Output, 2)
	assert.Equal(t, root.Children[0].Children[0].Output, 3)
}

func TestTimeIsMonotonic(t *testing.T) {
	ctx := context.Background()
	root := NewNode(func(ctx context.Context, input interface{}) interface{} {
		return 1
	}, nil)
	root.AddChild(func(ctx context.Context, input interface{}) interface{} {
		return 2
	}).AddChild(func(ctx context.Context, input interface{}) interface{} {
		return 3
	})
	root.Execute(ctx)
	assert.Assert(t, root.Started.Before(root.Ended))
	assert.Assert(t, root.Children[0].Started.Before(root.Children[0].Ended))
	assert.Assert(t, root.Children[0].Children[0].Started.Before(root.Children[0].Children[0].Ended))
}
