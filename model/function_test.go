package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Make sure that Check actually catches problems
func TestFuncBadCheck(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	v0 := &Variable{0, "V0", 0, 0, []float64{}}
	v1 := &Variable{1, "V1", 1, 0, []float64{1.0}}
	v2 := &Variable{2, "V2", 2, 0, []float64{0.25, 0.75}}
	v3 := &Variable{3, "V2", 3, 0, []float64{0.25, 0.70, 0.05}}

	// quick sanity check
	assert.NoError(v0.Check())
	assert.NoError(v1.Check())
	assert.NoError(v2.Check())
	assert.NoError(v3.Check())

	cases := []*Function{
		{"Bad-NoVarHaveTable", []*Variable{}, []float64{0.5, 0.5}, false},
		{"Bad-0Var", []*Variable{v0}, []float64{}, false},

		{"Bad-1Var1BadTable", []*Variable{v1}, []float64{0.5, 0.5}, false},
		{"Bad-1Var2BadTable", []*Variable{v2}, []float64{0.5, 0.5, 0.5}, false},
		{"Bad-1Var3BadTable", []*Variable{v3}, []float64{0.5}, false},

		{"Bad-2VarBadVar", []*Variable{v2, v0}, []float64{0.5, 0.5}, false},
		{"Bad-2VarBadTableHi", []*Variable{v2, v2}, []float64{0.5, 0.5, 0.5, 0.5, 0.5}, false},
		{"Bad-2VarBadTableLo", []*Variable{v2, v2}, []float64{0.5, 0.5}, false},

		{"Bad-3VarBadTable", []*Variable{v1, v2, v3}, []float64{0.5, 0.5, 0.5, 0.5, 0.5}, false},
	}

	for _, f := range cases {
		assert.Error(f.Check())
	}
}

// Make sure we're OK with valid functions
func TestFuncGoodCheck(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	v1 := &Variable{0, "V1", 1, 0, []float64{1.0}}
	v2 := &Variable{1, "V2", 2, 0, []float64{0.25, 0.75}}
	v3 := &Variable{2, "V2", 3, 0, []float64{0.25, 0.70, 0.05}}

	// quick sanity check
	assert.NoError(v1.Check())
	assert.NoError(v2.Check())
	assert.NoError(v3.Check())

	cases := []*Function{
		{"Good-1Var1", []*Variable{v1}, []float64{0.5}, false},
		{"Good-1Var2", []*Variable{v2}, []float64{0.5, 0.5}, false},
		{"Good-1Var3", []*Variable{v3}, []float64{0.5, 0.5, 0.5}, false},

		{"Good-2VarBin", []*Variable{v2, v2}, []float64{0.5, 0.5, 0.5, 0.5}, false},
		{"Good-2VarMad", []*Variable{v2, v3}, []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}, false},

		{"Good-3VarAll", []*Variable{v1, v2, v3}, []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}, false},
		{"Good-3VarNo1", []*Variable{v3, v2, v2}, []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}, false},
	}

	for _, f := range cases {
		assert.NoError(f.Check())
	}
}

// Make sure we correctly handle bad eval
func TestFuncTestEval(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	v2 := &Variable{0, "V2", 2, 0, []float64{0.25, 0.75}}
	v3 := &Variable{1, "V2", 3, 0, []float64{0.25, 0.70, 0.05}}

	// quick sanity check
	assert.NoError(v2.Check())
	assert.NoError(v3.Check())

	f := &Function{
		"TestTable",
		[]*Variable{v2, v3},
		[]float64{
			0.01, // 0 0
			1.02, // 0 1
			2.03, // 0 2
			3.04, // 1 0
			4.05, // 1 1
			5.06, // 1 2
		},
		false,
	}

	passCases := []struct {
		values   []int
		expected float64
	}{
		{[]int{0, 0}, 0.01},
		{[]int{0, 1}, 1.02},
		{[]int{0, 2}, 2.03},
		{[]int{1, 0}, 3.04},
		{[]int{1, 1}, 4.05},
		{[]int{1, 2}, 5.06},
	}

	failCases := [][]int{
		{},
		{0},
		{0, 0, 0},
		{2, 0},
		{0, 3},
	}

	const EPS float64 = 1e-14

	totProduct := 1.0

	for _, c := range passCases {
		v, e := f.Eval(c.values)
		assert.NoError(e)
		assert.InEpsilon(c.expected, v, EPS)
		totProduct *= v
	}

	for _, c := range failCases {
		v, e := f.Eval(c)
		assert.Error(e)
		assert.True(math.IsNaN(v))
	}

	// Now we can check our log space work
	assert.False(f.IsLog)
	assert.NoError(f.UseLogSpace())
	assert.True(f.IsLog)
	assert.Error(f.UseLogSpace())
	assert.True(f.IsLog)

	logSum := 0.0
	for _, c := range passCases {
		v, e := f.Eval(c.values)
		assert.NoError(e)
		logSum += v
	}

	assert.InEpsilon(totProduct, math.Exp(logSum), EPS)
}
