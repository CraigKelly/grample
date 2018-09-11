package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Make sure that Check actually catches problems
func TestVarBadCheck(t *testing.T) {
	assert := assert.New(t)

	// easy check
	_, e := NewVariable(1, 0)
	assert.Error(e)
	_, e = NewVariable(-1, 2)
	assert.Error(e)

	// bad cases
	cases := []Variable{
		{0, "BadVar-NoCardHaveMarg", 0, -1, []float64{0.5, 0.5}, nil, false},
		{1, "BadVar-HaveCardNoMarg", 2, -1, []float64{}, nil, false},
		{2, "BadVar-MismatchCardMarg", 2, -1, []float64{0.3, 0.3, 0.4}, nil, false},
		{3, "BadVar-MargNotADist<1", 2, -1, []float64{0.5, 0.4999}, nil, false},
		{4, "BadVar-MargNotADist>1", 2, -1, []float64{0.5, 0.5001}, nil, false},
		{5, "BadVar-InvalidFixVal", 2, -2, []float64{0.5, 0.5001}, nil, false},
		{5, "BadVar-FixVal>Card", 2, 3, []float64{0.5, 0.5001}, nil, false},
	}

	for _, v := range cases {
		assert.Error(v.Check())
	}
}

// Make sure that we can actually pass our tests
func TestVarGoodCheck(t *testing.T) {
	assert := assert.New(t)

	// Easy checks
	for i := 1; i <= 3; i++ {
		v, e := NewVariable(i, i)
		assert.NoError(e)
		assert.NoError(v.Check())
		assert.Equal(i, v.Card)
		assert.Equal(i, v.ID)
	}

	// good cases
	cases := []Variable{
		{0, "GoodVar-NoCard", 0, -1, []float64{}, nil, false},
		{1, "GoodVar-Card1", 1, -1, []float64{1.0}, nil, false},
		{2, "GoodVar-Card2", 2, -1, []float64{0.5, 0.5}, nil, false},
		{3, "GoodVar-Card3", 3, -1, []float64{0.5, 0.4, 0.1}, nil, false},
		{4, "GoodVar-Card3Fix", 3, 0, []float64{0.5, 0.4, 0.1}, nil, false},
		{5, "GoodVar-Card3Fix", 3, 2, []float64{0.5, 0.4, 0.1}, nil, false},
	}

	for _, v := range cases {
		assert.NoError(v.Check())
	}
}

// normalizing marginal testing
func TestVarNormProb(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Success bool
		Var     *Variable
	}{
		{false, &Variable{0, "BadVar-NoCardHaveMarg", 0, -1, []float64{0.5, 0.5}, nil, false}},
		{true, &Variable{1, "GoodVar-NoCard", 0, -1, []float64{}, nil, false}},
		{true, &Variable{2, "GoodVar-Card1-OK", 1, -1, []float64{1.0}, nil, false}},
		{true, &Variable{3, "GoodVar-Card1-SUB", 1, -1, []float64{0.1}, nil, false}},
		{true, &Variable{4, "GoodVar-Card2-OK", 2, -1, []float64{0.5, 0.5}, nil, false}},
		{true, &Variable{5, "GoodVar-Card2-SUB", 2, -1, []float64{120.0, 120.0}, nil, false}},
	}

	for _, c := range cases {
		if c.Success {
			assert.NoError(c.Var.NormMarginal())
			assert.NoError(c.Var.Check())
		} else {
			assert.Error(c.Var.NormMarginal())
			assert.Error(c.Var.Check())
		}
	}
}

// test our naming helper
func TestVarNaming(t *testing.T) {
	assert := assert.New(t)

	v := &Variable{0, "StartName", 0, -1, []float64{}, nil, false}

	assert.Error(v.CreateName(-1)) // Quick error testing

	cases := []struct {
		index int
		name  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{(26 * 26) + 26 - 1, "ZZ"},
		{(26 * 26) + 26, "AAA"},
	}

	for _, c := range cases {
		assert.NotEqual(c.name, v.Name)
		assert.NoError(v.CreateName(c.index))
		assert.Equal(c.name, v.Name)
	}
}

// test cloning
func TestVarClone(t *testing.T) {
	assert := assert.New(t)

	v1 := &Variable{1, "StartName", 2, -1, []float64{1.0, 2.1}, map[string]float64{"Abc": 42.42}, true}
	v2 := v1.Clone()
	assert.True(v1 != v2) // point to different objects
	assert.Equal(v1, v2)  // look exactly the same

	f1 := fmt.Sprintf("%+v", v1)
	f2 := fmt.Sprintf("%+v", v2)
	assert.Equal(f1, f2)
}
