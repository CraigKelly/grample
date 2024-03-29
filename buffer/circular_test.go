package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCircularInt(t *testing.T) {
	assert := assert.New(t)

	ci := NewCircularInt(6)
	assert.Equal(6, ci.BufSize)
	assert.Equal(0, ci.Count)

	assert.NoError(ci.Add(1))
	assert.NoError(ci.Add(2))
	assert.NoError(ci.Add(3))
	assert.NoError(ci.Add(4))
	assert.NoError(ci.Add(5))
	assert.Equal(6, ci.BufSize)
	assert.Equal(5, ci.Count)
	assert.Nil(ci.FirstHalf())
	assert.Nil(ci.SecondHalf())

	assert.NoError(ci.Add(6))
	assert.Equal(6, ci.BufSize)
	assert.Equal(6, ci.Count)

	exp := 0
	for iter := ci.FirstHalf(); iter.Next(); {
		val := iter.Value()
		exp++
		assert.Equal(exp, val)
	}
	for iter := ci.SecondHalf(); iter.Next(); {
		val := iter.Value()
		exp++
		assert.Equal(exp, val)
	}

	// 1 2 3 4 5 6 add 8 add 8 => 8 8 3 4 5 6
	// So first=3,4,5 second=6,8,8
	assert.NoError(ci.Add(8))
	assert.NoError(ci.Add(8))
	expVals := []int{3, 4, 5, 6, 8, 8}
	idx := 0
	for iter := ci.FirstHalf(); iter.Next(); {
		val := iter.Value()
		exp := expVals[idx]
		idx++
		assert.Equal(exp, val)
	}
	for iter := ci.SecondHalf(); iter.Next(); {
		val := iter.Value()
		exp := expVals[idx]
		idx++
		assert.Equal(exp, val)
	}
}
