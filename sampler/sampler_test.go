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

	i, e = uni.UniSample(0)
	assert.Error(e)

	i, e = uni.UniSample((1 << 30) + 1)
	assert.Error(e)

	i, e = uni.UniSample(1)
	assert.NoError(e)
	assert.Equal(0, i)

	vars := []*model.Variable{}
	i, e = uni.VarSample(vars, false)
	assert.Error(e)

	v1.Collapsed = true
	vars = []*model.Variable{v1}
	i, e = uni.VarSample(vars, true)
	assert.Error(e)
	i, e = uni.VarSample(vars, false)
	assert.NoError(e)
	assert.Equal(0, i)

	headCount := 0
	tailCount := 0
	flipCount := 0
	vars = []*model.Variable{v1, v2}

	for headCount < 1 || tailCount < 1 {
		i, e := uni.VarSample(vars, false)
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
	i, e = uni.VarSample(vars, false)
	assert.NoError(e)
	assert.Equal(1, i)

	// Fix v2, so there are no choices - that's an error
	v2.FixedVal = 1
	i, e = uni.VarSample(vars, false)
	assert.Error(e)

	// Now check with collapsed
	v1.FixedVal = -1
	v2.FixedVal = -1
	v1.Collapsed = true
	i, e = uni.VarSample(vars, true)
	assert.NoError(e)
	assert.Equal(1, i)
	v2.Collapsed = true
	i, e = uni.VarSample(vars, true)
	assert.Error(e)
}

func TestWeightedSampler(t *testing.T) {
	assert := assert.New(t)

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)
	uni, err := NewUniformSampler(gen, 32)
	assert.NoError(err)

	var i int
	var e error

	i, e = uni.WeightedSample(0, []float64{})
	assert.Error(e)

	i, e = uni.WeightedSample((1<<30)+1, []float64{})
	assert.Error(e)

	i, e = uni.WeightedSample(1, []float64{})
	assert.Error(e)
	i, e = uni.WeightedSample(1, []float64{1.0, 1.0})
	assert.Error(e)

	i, e = uni.WeightedSample(2, []float64{1.0, -1.0})
	assert.Error(e)

	i, e = uni.WeightedSample(1, []float64{1.0})
	assert.NoError(e)
	assert.Equal(0, i)

	weights := []float64{100.1, 200.2}
	headCount := 0.0
	tailCount := 0.0
	flipCount := 0
	for headCount < 100.0 || tailCount < 100.0 {
		i, e := uni.WeightedSample(2, weights)
		assert.NoError(e)
		assert.True(i >= 0 && i <= 1)
		if i == 0 {
			headCount += 1.0
		} else if i == 1 {
			tailCount += 1.0
		} else {
			assert.True(false, "TEST BUG: how did this happen?")
		}

		flipCount++
		if flipCount > 5000 {
			assert.True(false, "TEST BUG: too many flips %v (%v / %v)", flipCount, headCount, tailCount)
			break
		}
	}

	assert.InEpsilon(0.5, headCount/tailCount, 0.1)
}
