package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/config"
	"gotest.tools/assert"
)

func TestNewBacalhauExecutor(t *testing.T) {
	e := NewBacalhauExecutor()
	assert.Assert(t, e != nil)
}

func TestBacalhauExecutor_Execute(t *testing.T) {
	t.Skip("Can't be bothered to do this until Bacalhau has an API interface.")
}

func TestBacalhauExecutor_ErrorWhenEmptyCID(t *testing.T) {
	e := NewBacalhauExecutor()
	_, err := e.Render(config.Job{}, []ExecutorIOSpec{
		{
			Name: "test-input",
			Ref:  "",
			Path: "/",
		},
	}, []ExecutorIOSpec{})
	assert.ErrorContains(t, err, "input CID for")
}

func TestBacalhauRendersProperly(t *testing.T) {
	ctx := context.Background()
	b := NewBacalhauExecutor()
	j, err := b.Render(config.Job{
		ID:         "test-job",
		Image:      "ubuntu:latest",
		Entrypoint: []string{"echo", "ok"},
	}, []ExecutorIOSpec{}, []ExecutorIOSpec{})
	assert.NilError(t, err)
	result, err := b.Execute(ctx, j)
	fmt.Println(result.ID)
	assert.NilError(t, err)
}
