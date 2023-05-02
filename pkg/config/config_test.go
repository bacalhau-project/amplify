package config

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetConfig(t *testing.T) {
	c, err := GetConfig("../../config.yaml")
	assert.NilError(t, err)
	assert.Assert(t, c != nil)
	assert.Assert(t, len(c.Jobs) > 0)
	assert.Assert(t, len(c.Jobs[0].ID) > 0)
	assert.Assert(t, len(c.Graph) > 0)
	assert.Assert(t, len(c.Graph[0].ID) > 0)

	assert.Equal(t, c.Jobs[0].Memory, DefaultMemory)
	assert.Equal(t, c.Jobs[0].CPU, DefaultCPU)
	assert.Equal(t, c.Jobs[0].Timeout, DefaultTimeout)

	_, err = GetConfig("nonexistent.yaml")
	assert.Assert(t, err != nil)
}
