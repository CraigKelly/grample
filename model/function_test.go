package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testVars() (v0, v1, v2, v3 *Variable) {
	v0 = &Variable{0, "V0", 0, -1, []float64{}, nil, false}
	v1 = &Variable{1, "V1", 1, -1, []float64{1.0}, nil, false}
	v2 = &Variable{2, "V2", 2, -1, []float64{0.25, 0.75}, nil, false}
	v3 = &Variable{3, "V2", 3, -1, []float64{0.25, 0.70, 0.05}, nil, false}
	return
}

// Make sure that Check actually catches problems
func TestFuncBadCheck(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	v0, v1, v2, v3 := testVars()

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
	_, v1, v2, v3 := testVars()

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
	_, _, v2, v3 := testVars()

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

// test cloning
func TestFuncClone(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	_, _, v2, v3 := testVars()
	v2 = v2.Clone()
	v3 = v3.Clone()

	// quick sanity check
	assert.NoError(v2.Check())
	assert.NoError(v3.Check())

	f1 := &Function{
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

	f2 := f1.Clone()
	assert.True(f1 != f2) // point to different objects
	assert.Equal(f1, f2)  // look exactly the same
}

// test function creation
func TestFuncBuildup(t *testing.T) {
	assert := assert.New(t)

	// handy short vars for below
	_, _, v2, v3 := testVars()
	f, err := NewFunction(0, []*Variable{v2, v3})
	assert.NoError(err)

	// Add 1 for every variable configuration
	vi, err := NewVariableIter(f.Vars)
	assert.NoError(err)
	vals := make([]int, len(f.Vars))
	for {
		err := vi.Val(vals)
		assert.NoError(err)
		if err != nil {
			return
		}

		f.AddValue(vals, 1.0)

		if !vi.Next() {
			break
		}
	}

	// Should have a 2x3 table with every entry at 1.0
	assert.Equal(6, len(f.Table))
	for _, v := range f.Table {
		assert.InEpsilon(1.0, v, 1e-6)
	}

	// Now add some more and recheck
	vi, err = NewVariableIter(f.Vars)
	assert.NoError(err)
	for {
		err := vi.Val(vals)
		assert.NoError(err)
		if err != nil {
			return
		}

		f.AddValue(vals, 2.42)

		if !vi.Next() {
			break
		}
	}

	// 1 + 2.42 = 3.42
	for _, v := range f.Table {
		assert.InEpsilon(3.42, v, 1e-6)
	}

	// Now make sure that log space addition fails and doesn't break anything
	// (and don't forget we're in log space when checking values :)
	assert.NoError(f.UseLogSpace())
	assert.Error(f.AddValue([]int{0, 0}, 123.45))
	assert.InEpsilon(math.Log(3.42), f.Table[0], 1e-6)
}
