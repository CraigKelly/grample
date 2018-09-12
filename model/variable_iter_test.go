package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVarIter(t *testing.T) {
	assert := assert.New(t)

	v1, e := NewVariable(1, 2)
	assert.NoError(e)
	v2, e := NewVariable(2, 3)
	assert.NoError(e)
	v3, e := NewVariable(3, 2)
	assert.NoError(e)

	vi, e := NewVariableIter([]*Variable{v1, v2, v3})
	assert.NoError(e)

	expected := [][]int{
		[]int{0, 0, 0},
		[]int{0, 0, 1},
		[]int{0, 1, 0},
		[]int{0, 1, 1},
		[]int{0, 2, 0},
		[]int{0, 2, 1},
		[]int{1, 0, 0},
		[]int{1, 0, 1},
		[]int{1, 1, 0},
		[]int{1, 1, 1},
		[]int{1, 2, 0},
		[]int{1, 2, 1},
	}

	vals := make([]int, 3)
	curr := 0
	for {
		assert.NoError(vi.Val(vals))
		assert.Equal(expected[curr], vals)
		if !vi.Next() {
			break
		}
		curr++
	}

	assert.Equal([]int{0, 0, 0}, vi.lastVal)
}

func TestVarIterCorners(t *testing.T) {
	assert := assert.New(t)

	// Creation error
	vi, e := NewVariableIter([]*Variable{})
	assert.Error(e)
	vi, e = NewVariableIter(nil)
	assert.Error(e)

	v, e := NewVariable(0, 2)
	assert.NoError(e)
	vi, e = NewVariableIter([]*Variable{v})
	assert.NoError(e)

	// Value error
	vals := []int{}
	assert.Error(vi.Val(vals))

	// Working single var loop with oversized slice
	vals = []int{0, 0}
	assert.NoError(vi.Val(vals))
	assert.Equal([]int{0, 0}, vals)
	assert.True(vi.Next())
	assert.NoError(vi.Val(vals))
	assert.Equal([]int{1, 0}, vals)
	assert.False(vi.Next())
}
