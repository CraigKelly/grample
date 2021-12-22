package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func checkExpectSeq(assert *assert.Assertions, fixed bool, vars []*Variable, expected [][]int) {
	vi, e := NewVariableIter(vars, fixed)
	assert.NoError(e)

	vals := make([]int, len(vars))
	curr := 0
	for {
		assert.NoError(vi.Val(vals))
		assert.Equal(expected[curr], vals)
		if !vi.Next() {
			break
		}
		curr++
	}

	assert.Equal(len(expected)-1, curr)

	// Iterators wrap back around to the start value, which is different if
	// there are fixed values
	finalVal := make([]int, len(vars))
	if fixed {
		for i, v := range vars {
			if v.FixedVal >= 0 {
				finalVal[i] = v.FixedVal
			}
		}
	}

	assert.Equal(finalVal, vi.lastVal)
}

func TestVarIter(t *testing.T) {
	assert := assert.New(t)

	v1, e := NewVariable(1, 2)
	assert.NoError(e)
	v2, e := NewVariable(2, 3)
	assert.NoError(e)
	v3, e := NewVariable(3, 2)
	assert.NoError(e)

	checkExpectSeq(assert, false, []*Variable{v1, v2, v3}, [][]int{
		{0, 0, 0},
		{0, 0, 1},
		{0, 1, 0},
		{0, 1, 1},
		{0, 2, 0},
		{0, 2, 1},
		{1, 0, 0},
		{1, 0, 1},
		{1, 1, 0},
		{1, 1, 1},
		{1, 2, 0},
		{1, 2, 1},
	})
}

func TestVarIterCorners(t *testing.T) {
	assert := assert.New(t)

	// Creation error
	_, e := NewVariableIter([]*Variable{}, false)
	assert.Error(e)
	_, e = NewVariableIter(nil, false)
	assert.Error(e)

	v, e := NewVariable(0, 2)
	assert.NoError(e)
	vi, e := NewVariableIter([]*Variable{v}, false)
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

func TestVarIterFixedVals(t *testing.T) {
	assert := assert.New(t)
	var e error

	// All fixed
	v1, e := NewVariable(0, 2)
	assert.NoError(e)
	v2, e := NewVariable(1, 2)
	assert.NoError(e)

	v1.FixedVal = 1
	v2.FixedVal = 0
	vi, e := NewVariableIter([]*Variable{v1, v2}, true)
	assert.NoError(e)
	vals := []int{0, 0}
	assert.NoError(vi.Val(vals))
	assert.Equal([]int{1, 0}, vals)
	assert.False(vi.Next())

	// One fixed
	v1.FixedVal = -1
	v2.FixedVal = -1
	vFix, e := NewVariable(2, 2)
	vFix.FixedVal = 1
	assert.NoError(e)

	checkExpectSeq(assert, true, []*Variable{vFix, v1, v2}, [][]int{
		{1, 0, 0},
		{1, 0, 1},
		{1, 1, 0},
		{1, 1, 1},
	})
	checkExpectSeq(assert, true, []*Variable{v1, vFix, v2}, [][]int{
		{0, 1, 0},
		{0, 1, 1},
		{1, 1, 0},
		{1, 1, 1},
	})
	checkExpectSeq(assert, true, []*Variable{v1, v2, vFix}, [][]int{
		{0, 0, 1},
		{0, 1, 1},
		{1, 0, 1},
		{1, 1, 1},
	})
}
