package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func vanillaModel() *Model {
	v1 := &Variable{0, "V1", 2, -1, []float64{0.5, 0.5}, nil, false}
	v2 := &Variable{1, "V2", 2, -1, []float64{0.5, 0.5}, nil, false}

	f1 := &Function{"F1", []*Variable{v1, v2}, []float64{1.1, 2.2, 3.3, 4.4}, false}
	f2 := &Function{"F2", []*Variable{v1, v2}, []float64{0.1, 0.2, 0.3, 0.4}, false}

	return &Model{
		Type:  "MARKOV",
		Name:  "TestingModel",
		Vars:  []*Variable{v1, v2},
		Funcs: []*Function{f1, f2},
	}
}

func TestModelCreation(t *testing.T) {
	assert := assert.New(t)

	const eps = 1e-8

	// Make sure we have a valid model that looks like we expect before we start breaking things
	m := vanillaModel()
	assert.NoError(m.Check())
	assert.Equal(MARKOV, m.Type)
	assert.Equal(0, m.Vars[0].ID)
	assert.Equal(1, m.Vars[1].ID)

	var v float64
	var e error

	v, e = m.Funcs[0].Eval([]int{1, 1})
	assert.NoError(e)
	assert.InEpsilon(4.4, v, eps)

	v, e = m.Funcs[1].Eval([]int{1, 1})
	assert.NoError(e)
	assert.InEpsilon(0.4, v, eps)

	// Check dup ID
	m.Vars[0].ID = 1
	assert.Error(m.Check())

	m = vanillaModel()
	m.Type = "NOPE"
	assert.Error(m.Check())

	m = vanillaModel()
	m.Vars[0].Card = 0
	assert.Error(m.Check())

	m = vanillaModel()
	m.Funcs[0].Table = []float64{0.0}
	assert.Error(m.Check())
}
