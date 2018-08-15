package sampler

import (
	"testing"

	"github.com/CraigKelly/grample/model"

	"github.com/stretchr/testify/assert"
)

func TestMergeChains(t *testing.T) {
	assert := assert.New(t)

	var vars []*model.Variable
	var err error

	vars, err = MergeChains([]*Chain{})
	assert.Nil(vars)
	assert.Error(err)

	v1 := &model.Variable{ID: 0, Card: 2, FixedVal: -1, Marginal: []float64{0.5, 0.5}}
	v2 := &model.Variable{ID: 1, Card: 2, FixedVal: -1, Marginal: []float64{5.1, 5.1}}
	v3 := &model.Variable{ID: 2, Card: 3, FixedVal: -1, Marginal: []float64{1.1, 2.2, 3.3}}

	mod := &model.Model{
		Type:  "MARKOV",
		Name:  "TestingModel",
		Vars:  []*model.Variable{v1, v2, v3},
		Funcs: nil,
	}

	ch1, err := NewChain(mod.Clone(), nil, 0, 0)
	assert.NoError(err)

	vars, err = MergeChains([]*Chain{ch1, ch1})
	assert.NoError(err)
	assert.InDeltaSlice([]float64{1.0, 1.0}, vars[0].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{10.2, 10.2}, vars[1].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{2.2, 4.4, 6.6}, vars[2].Marginal, 1e-8)

	// Test 1-chain merge after so that we know the variables weren't changed
	vars, err = MergeChains([]*Chain{ch1})
	assert.NoError(err)
	assert.InDeltaSlice([]float64{0.5, 0.5}, vars[0].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{5.1, 5.1}, vars[1].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{1.1, 2.2, 3.3}, vars[2].Marginal, 1e-8)
}
