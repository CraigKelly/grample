package sampler

import (
	"fmt"
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

	v1 := &model.Variable{ID: 0, Card: 2, FixedVal: -1, Marginal: []float64{0.5, 0.5}, Collapsed: false}
	v2 := &model.Variable{ID: 1, Card: 2, FixedVal: -1, Marginal: []float64{5.1, 5.1}, Collapsed: false}
	v3 := &model.Variable{ID: 2, Card: 3, FixedVal: -1, Marginal: []float64{1.1, 2.2, 3.3}, Collapsed: false}

	mod := &model.Model{
		Type:  "MARKOV",
		Name:  "TestingModel",
		Vars:  []*model.Variable{v1, v2, v3},
		Funcs: nil,
	}

	ch1, err := NewChain(mod.Clone(), nil, 0, 0)
	assert.NoError(err)

	// Test 1-chain merge after so that we know the variables weren't changed
	type Chains []*Chain
	oneVarTest := func(chs Chains) {
		vars, err := MergeChains(chs)
		assert.NoError(err)
		assert.InDeltaSlice([]float64{0.5, 0.5}, vars[0].Marginal, 1e-8)
		assert.InDeltaSlice([]float64{5.1, 5.1}, vars[1].Marginal, 1e-8)
		assert.InDeltaSlice([]float64{1.1, 2.2, 3.3}, vars[2].Marginal, 1e-8)
	}

	// Multi chain merge
	vars, err = MergeChains(Chains{ch1, ch1})
	assert.NoError(err)
	assert.InDeltaSlice([]float64{1.0, 1.0}, vars[0].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{10.2, 10.2}, vars[1].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{2.2, 4.4, 6.6}, vars[2].Marginal, 1e-8)

	oneVarTest(Chains{ch1}) // make sure no vars changed

	// Now test to make sure that collapsing works
	v1.Collapsed = true
	ch2, err := NewChain(mod.Clone(), nil, 0, 0)
	assert.NoError(err)
	fmt.Printf("%+v\n", mod.Vars[0])
	fmt.Printf("%+v\n", ch2.Target.Vars[0])

	vars, err = MergeChains(Chains{ch1, ch2})
	assert.NoError(err)
	assert.InDeltaSlice([]float64{0.5, 0.5}, vars[0].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{10.2, 10.2}, vars[1].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{2.2, 4.4, 6.6}, vars[2].Marginal, 1e-8)

	vars, err = MergeChains(Chains{ch2, ch1})
	assert.NoError(err)
	assert.InDeltaSlice([]float64{0.5, 0.5}, vars[0].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{10.2, 10.2}, vars[1].Marginal, 1e-8)
	assert.InDeltaSlice([]float64{2.2, 4.4, 6.6}, vars[2].Marginal, 1e-8)

	oneVarTest(Chains{ch1})
	oneVarTest(Chains{ch2})

	v2.Collapsed = true
	v3.Collapsed = true
	ch3, err := NewChain(mod.Clone(), nil, 0, 0)
	assert.NoError(err)
	oneVarTest(Chains{ch1, ch2, ch3}) // all collapsed means should act one single chain
	oneVarTest(Chains{ch1})           // Make sure original chain still OK
}
