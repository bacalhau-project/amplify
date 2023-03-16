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
	assert.Assert(t, len(c.Jobs[0].Name) > 0)
	assert.Assert(t, len(c.Workflows) > 0)
	assert.Assert(t, len(c.Workflows[0].Name) > 0)

	_, err = GetConfig("nonexistent.yaml")
	assert.Assert(t, err != nil)
}
