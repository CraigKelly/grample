package model

import (
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
		{0, "BadVar-NoCardHaveMarg", 0, []float64{0.5, 0.5}, nil},
		{1, "BadVar-HaveCardNoMarg", 2, []float64{}, nil},
		{2, "BadVar-MismatchCardMarg", 2, []float64{0.3, 0.3, 0.4}, nil},
		{3, "BadVer-MargNotADist<1", 2, []float64{0.5, 0.4999}, nil},
		{4, "BadVer-MargNotADist>1", 2, []float64{0.5, 0.5001}, nil},
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
		{0, "GoodVar-NoCard", 0, []float64{}, nil},
		{1, "GoodVar-Card1", 1, []float64{1.0}, nil},
		{2, "GoodVar-Card2", 2, []float64{0.5, 0.5}, nil},
		{3, "GoodVar-Card3", 3, []float64{0.5, 0.4, 0.1}, nil},
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
		{false, &Variable{0, "BadVar-NoCardHaveMarg", 0, []float64{0.5, 0.5}, nil}},
		{true, &Variable{1, "GoodVar-NoCard", 0, []float64{}, nil}},
		{true, &Variable{2, "GoodVar-Card1-OK", 1, []float64{1.0}, nil}},
		{true, &Variable{3, "GoodVar-Card1-SUB", 1, []float64{0.1}, nil}},
		{true, &Variable{4, "GoodVar-Card2-OK", 2, []float64{0.5, 0.5}, nil}},
		{true, &Variable{5, "GoodVar-Card2-SUB", 2, []float64{120.0, 120.0}, nil}},
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

	v := &Variable{0, "StartName", 0, []float64{}, nil}

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
