package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorSuite(t *testing.T) {
	assert := assert.New(t)

	vars1 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{250.0, 750.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{25.1, 75.3}, nil},
	}
	vars2 := []*Variable{
		&Variable{0, "V1", 2, -1, []float64{42.0, 42.0}, nil},
		&Variable{0, "V2", 2, -1, []float64{3.1, 3.1}, nil},
	}

	// Calculate mean hellinger
	p1 := math.Pow(math.Sqrt(0.75)-math.Sqrt(0.50), 2)
	p2 := math.Pow(math.Sqrt(0.25)-math.Sqrt(0.50), 2)
	hellExp := math.Sqrt(p1+p2) / math.Sqrt2

	/* JS Divergence calc via python with from scipy.stats import entropy
	from numpy.linalg import norm
	import numpy as np
	def jsd(p, q):
		_p = p / norm(p, ord=1)
		_q = q / norm(q, ord=1)
		_m = 0.5 * (_p + _q)
		return 0.5 * (entropy(_p, _m, base=2) + entropy(_q, _m, base=2))
	print(jsd([0.5, 0.5], [0.25, 0.75]))
	*/
	jsExp := 0.0487949406953985

	var suite *ErrorSuite
	var err error
	const eps = 1e-8

	// TODO: max also

	// 2 non-normed

	suite, err = NewErrorSuite(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.25, suite.MeanMeanAbsError, eps)
	assert.InEpsilon(0.25, suite.MeanMaxAbsError, eps)
	assert.InEpsilon(hellExp, suite.MeanHellinger, eps)
	assert.InEpsilon(jsExp, suite.MeanJSDiverge, eps)

	// 1 non-normed

	assert.NoError(vars1[0].NormMarginal())
	assert.NoError(vars2[1].NormMarginal())

	suite, err = NewErrorSuite(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.25, suite.MeanMeanAbsError, eps)
	assert.InEpsilon(0.25, suite.MeanMaxAbsError, eps)
	assert.InEpsilon(hellExp, suite.MeanHellinger, eps)
	assert.InEpsilon(jsExp, suite.MeanJSDiverge, eps)

	// All normed
	assert.NoError(vars1[1].NormMarginal())
	assert.NoError(vars2[0].NormMarginal())

	suite, err = NewErrorSuite(vars1, vars2)
	assert.NoError(err)
	assert.InEpsilon(0.25, suite.MeanMeanAbsError, eps)
	assert.InEpsilon(0.25, suite.MeanMaxAbsError, eps)
	assert.InEpsilon(hellExp, suite.MeanHellinger, eps)
	assert.InEpsilon(jsExp, suite.MeanJSDiverge, eps)
}
