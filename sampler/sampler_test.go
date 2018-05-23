package sampler

import (
	"fmt"
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"

	"github.com/stretchr/testify/assert"
)

func testVars() (v1 *model.Variable, v2 *model.Variable) {
	var e error

	v1, e = model.NewVariable(0, 2)
	if e != nil {
		panic(fmt.Sprintf("%v", e))
	}

	v2, e = model.NewVariable(1, 2)
	if e != nil {
		panic(fmt.Sprintf("%v", e))
	}

	return
}

func TestUniformSampler(t *testing.T) {
	assert := assert.New(t)

	v1, v2 := testVars()

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)
	uni, err := NewUniformSampler(gen, 32)
	assert.NoError(err)

	var i int
	var e error

	i, e = uni.ValSample(0)
	assert.Error(e)

	i, e = uni.ValSample(1)
	assert.NoError(e)
	assert.Equal(0, i)

	vars := []*model.Variable{}
	i, e = uni.VarSample(vars)
	assert.Error(e)

	vars = []*model.Variable{v1}
	i, e = uni.VarSample(vars)
	assert.NoError(e)
	assert.Equal(0, i)

	headCount := 0
	tailCount := 0
	flipCount := 0
	vars = []*model.Variable{v1, v2}

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

func TestUniformSamplerFixed(t *testing.T) {
	assert := assert.New(t)

	v1, v2 := testVars()

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)
	uni, err := NewUniformSampler(gen, 32)
	assert.NoError(err)

	var i int
	var e error
	vars := []*model.Variable{v1, v2}

	// Fix v1, so selection must be v2
	v1.FixedVal = 0
	i, e = uni.VarSample(vars)
	assert.NoError(e)
	assert.Equal(1, i)

	// Fix v2, so there are no choices - that's an error
	v2.FixedVal = 1
	i, e = uni.VarSample(vars)
	assert.Error(e)
}
