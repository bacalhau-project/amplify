package executor

import (
	"testing"

	"gotest.tools/assert"
)

func TestNewBacalhauExecutor(t *testing.T) {
	e := NewBacalhauExecutor()
	assert.Assert(t, e != nil)
	// assert.Equal(t, e.GetClient())

	// BacalhauExecutor(e)
	// client := e.GetClient()
}

func TestBacalhauExecutor_Execute(t *testing.T) {
	t.Skip("Can't be bothered to do this until Bacalhau has an API interface.")
}
