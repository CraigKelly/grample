package sampler

import (
	"math/rand"
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/stretchr/testify/assert"
)

func TestUniformSampler(t *testing.T) {
	assert := assert.New(t)

	r := rand.NewSource(42)
	uni, err := NewUniformSampler(r)
	assert.NoError(err)

	var i int
	var e error

	i, e = uni.ValSample(0)
	assert.Error(e)

	i, e = uni.ValSample(1)
	assert.NoError(e)
	assert.Equal(0, i)

	vars := make([]*model.Variable, 0)
	i, e = uni.VarSample(vars)
	assert.Error(e)

	vars = make([]*model.Variable, 1)
	i, e = uni.VarSample(vars)
	assert.NoError(e)
	assert.Equal(0, i)

	headCount := 0
	tailCount := 0
	flipCount := 0
	vars = make([]*model.Variable, 2)

	for headCount < 1 || tailCount < 1 {
		i, e := uni.VarSample(vars)
		assert.NoError(e)
		assert.True(i >= 0 && i <= 1)
		if i == 0 {
			headCount++
		} else if i == 1 {
			tailCount++
		} else {
			assert.True(false, "TEST BUG: how did this happen?")
		}

		flipCount++
		if flipCount > 2500 {
			break // odds of this if everything is working are 2.66e-753
		}
	}

	assert.True(headCount > 0 && tailCount > 0, "Well, that seems unlikely, H=%d,T=%d over %d", headCount, tailCount, flipCount)
}
