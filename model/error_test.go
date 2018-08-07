package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbsMaxError(t *testing.T) {
	assert := assert.New(t)

	vars1 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{250.0, 750.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{25.1, 75.3}, nil},
	}
	vars2 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{42.0, 42.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{3.1, 3.1}, nil},
	}

	var totAE float64
	var maxAE float64
	var err error

	// 2 non-normed

	totAE, maxAE, err = AbsError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)

	totAE, maxAE, err = AbsError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)

	// 1 non-normed

	assert.NoError(vars1[0].NormMarginal())
	assert.NoError(vars2[1].NormMarginal())

	totAE, maxAE, err = AbsError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)

	totAE, maxAE, err = AbsError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)

	// All normed
	assert.NoError(vars1[1].NormMarginal())
	assert.NoError(vars2[0].NormMarginal())

	totAE, maxAE, err = AbsError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)

	totAE, maxAE, err = AbsError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(0.50, totAE, 1e-8)
	assert.InEpsilon(0.25, maxAE, 1e-8)
}

func TestHellingerError(t *testing.T) {
	assert := assert.New(t)

	vars1 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{25.0, 75.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{250.3, 750.9}, nil},
	}
	vars2 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{42.0, 42.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{3.6, 3.6}, nil},
	}

	var hell float64
	var err error

	// Hellinger we just calculate directly
	p1 := math.Pow(math.Sqrt(0.75)-math.Sqrt(0.50), 2)
	p2 := math.Pow(math.Sqrt(0.25)-math.Sqrt(0.50), 2)
	hellExp := (p1 + p2) / math.Sqrt2

	// 2 non-normed

	hell, err = HellingerError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)

	hell, err = HellingerError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)

	// 1 non-normed

	assert.NoError(vars1[0].NormMarginal())
	assert.NoError(vars2[1].NormMarginal())

	hell, err = HellingerError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)

	hell, err = HellingerError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)

	// All normed
	assert.NoError(vars1[1].NormMarginal())
	assert.NoError(vars2[0].NormMarginal())

	hell, err = HellingerError(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)

	hell, err = HellingerError(vars2, vars1)
	assert.NoError(err)
	assert.InEpsilon(hellExp, hell, 1e-8)
}
