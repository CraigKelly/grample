package model

import (
	"math"

	"github.com/pkg/errors"
)

// TODO: add Jensen-Shannon to metrics
// TODO: Create GR diagnostic, accepting any of our metrics
// TODO: better unit testing for these methods

// All error functions accept 2 variable arrays that must be of the same length

// AbsError returns both the total and max absolute error between the model's
// current marginal estimations and this solution. The final score is the mean
// over all variables. The solution marginal is assumed to be normalized, the
// model variables are NOT.
func AbsError(vars1 []*Variable, vars2 []*Variable) (absErrMean float64, maxErrMean float64, failed error) {
	if len(vars1) != len(vars1) {
		return math.NaN(), math.NaN(), errors.Errorf("Variable count mismatch %d != %d", len(vars1), len(vars1))
	}

	totErrSum := float64(0.0) // Total error
	totErrMax := float64(0.0) // Total MAX error (max per var)
	const eps = float64(1e-12)

	// Remember that we skip vars
	varCount := 0

	for i, v1 := range vars1 {
		v2 := vars2[i]
		if v1.FixedVal >= 0 || v2.FixedVal >= 0 {
			continue
		}

		varCount++

		card := v1.Card
		if card != v2.Card {
			return math.NaN(), math.NaN(), errors.Errorf("Variable %d card mismatch %d != %d", i, card, v2.Card)
		}

		// get totals for normalizing
		tot1, tot2 := float64(0.0), float64(0.0)
		for c := 0; c < card; c++ {
			tot1 += v1.Marginal[c]
			tot2 += v2.Marginal[c]
		}
		if tot1 < eps {
			tot1 = eps
		}
		if tot2 < eps {
			tot2 = eps
		}

		// accumulate error (normalizing model var)
		maxErr := float64(0.0)
		for c := 0; c < card; c++ {
			adjVal1 := v1.Marginal[c] / tot1
			adjVal2 := v2.Marginal[c] / tot2
			err := math.Abs(adjVal1 - adjVal2)

			totErrSum += err // Just accumulate for total error

			if c == 0 || err > maxErr {
				maxErr = err // Found the max error
			}
		}
		totErrMax += maxErr
	}

	if varCount < 1 {
		return math.NaN(), math.NaN(), errors.Errorf("No un-fixed vars found to score")
	}

	absErrMean = totErrSum / float64(varCount)
	maxErrMean = totErrMax / float64(varCount)
	return
}

// HellingerError returns the Hellinger error between the model's current
// marginal estimate and this solution. Like AbsError, the result is the
// average over the variables, the solution's marginals are assumed normalized
// (sum=1.0), while the model's marginals are assumed non-normalized (but
// positive)
func HellingerError(vars1 []*Variable, vars2 []*Variable) (float64, error) {
	if len(vars1) != len(vars2) {
		return math.NaN(), errors.Errorf("Solution var count %d != model var count %d", len(vars1), len(vars2))
	}

	totErr := float64(0.0)
	const eps = float64(1e-12)

	// No fixed vars
	varCount := 0

	for i, v1 := range vars1 {
		v2 := vars2[i]
		if v1.FixedVal >= 0 || v2.FixedVal >= 0 {
			continue
		}

		varCount++

		card := v1.Card
		if card != v2.Card {
			return math.NaN(), errors.Errorf("Variable %d card mismatch %d != %d", i, card, v2.Card)
		}

		// get totals for normalizing
		tot1, tot2 := float64(0.0), float64(0.0)
		for c := 0; c < card; c++ {
			tot1 += v1.Marginal[c]
			tot2 += v2.Marginal[c]
		}
		if tot1 < eps {
			tot1 = eps
		}
		if tot2 < eps {
			tot2 = eps
		}

		// accumulate error (normalizing model var). Hellinger distance is
		// similar to the Euclidean L2: sum((sqrt(p) - sqrt(q))**2) / sqrt(2)
		errSum := float64(0.0)
		for c := 0; c < card; c++ {
			adjVal1 := math.Sqrt(v1.Marginal[c] / tot1)
			adjVal2 := math.Sqrt(v2.Marginal[c] / tot2)
			err := math.Pow(adjVal1-adjVal2, 2) // squared, so always positive
			errSum += err
		}
		totErr += errSum / math.Sqrt2
	}

	if varCount < 1 {
		return math.NaN(), errors.Errorf("No un-fixed vars to score")
	}

	return totErr / float64(varCount), nil
}
