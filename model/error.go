package model

import (
	"math"

	"github.com/pkg/errors"
)

// TODO: ErrorSuite creation/method for GR diagnostic

// ErrorSuite represents all the loss/error functions we use to judge progress
// across joint dist. Errors beginning with Mean are the mean across all the
// variables in the joint distribution while Max is the maximum value for all
// the variables. So MeanMaxAbsError is the MEAN of the Maximum Absoulte Error
// for each of the marginal variables. Likewise, MaxMeanAbsError represents the
// maximum value of the mean difference between two random variables.
type ErrorSuite struct {
	MeanMeanAbsError float64
	MeanMaxAbsError  float64
	MeanHellinger    float64
	MeanJSDiverge    float64

	MaxMeanAbsError float64
	MaxMaxAbsError  float64
	MaxHellinger    float64
	MaxJSDiverge    float64
}

// NewErrorSuite returns an ErrorSuite with all calculated error functions
func NewErrorSuite(vars1 []*Variable, vars2 []*Variable) (*ErrorSuite, error) {
	if len(vars1) != len(vars1) {
		return nil, errors.Errorf("Variable count mismatch %d != %d", len(vars1), len(vars1))
	}

	varCount := 0
	for i, v1 := range vars1 {
		v2 := vars2[i]
		if v1.Card != v2.Card {
			return nil, errors.Errorf("Variable card mismatch %d != %d", v1.Card, v2.Card)
		}
		if v1.FixedVal < 0 && v2.FixedVal < 0 {
			varCount++
		}
	}

	if varCount < 1 {
		return nil, errors.Errorf("No un-fixed vars to score")
	}

	es := ErrorSuite{}

	var d float64
	for i, v1 := range vars1 {
		v2 := vars2[i]

		d = MeanAbsDiff(v1, v2)
		es.MeanMeanAbsError += d
		es.MaxMeanAbsError = math.Max(d, es.MaxMeanAbsError)

		d = MaxAbsDiff(v1, v2)
		es.MeanMaxAbsError += d
		es.MaxMaxAbsError = math.Max(d, es.MaxMaxAbsError)

		d = HellingerDiff(v1, v2)
		es.MeanHellinger += d
		es.MaxHellinger = math.Max(d, es.MaxHellinger)

		d = JSDivergence(v1, v2)
		es.MeanJSDiverge += d
		es.MaxJSDiverge = math.Max(d, es.MaxJSDiverge)
	}

	fc := float64(varCount)
	es.MeanMeanAbsError /= fc
	es.MeanMaxAbsError /= fc
	es.MeanHellinger /= fc
	es.MeanJSDiverge /= fc

	return &es, nil
}

// MaxAbsDiff returns the maximum difference found between the two prob dists
func MaxAbsDiff(v1 *Variable, v2 *Variable) float64 {
	card := v1.Card

	// get totals for normalizing
	tot1, tot2 := float64(0.0), float64(0.0)
	const eps = 1e-12

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

	maxErr := float64(0.0)
	for c := 0; c < card; c++ {
		adjVal1 := v1.Marginal[c] / tot1
		adjVal2 := v2.Marginal[c] / tot2
		err := math.Abs(adjVal1 - adjVal2)
		if c == 0 || err > maxErr {
			maxErr = err
		}
	}

	return maxErr
}

// MeanAbsDiff returns the mean of the differenced found between the two prob dists
func MeanAbsDiff(v1 *Variable, v2 *Variable) float64 {
	card := v1.Card

	if card < 1 {
		return 0
	}

	// get totals for normalizing
	tot1, tot2 := float64(0.0), float64(0.0)
	const eps = 1e-12

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

	errSum := float64(0.0)
	for c := 0; c < card; c++ {
		adjVal1 := v1.Marginal[c] / tot1
		adjVal2 := v2.Marginal[c] / tot2
		errSum += math.Abs(adjVal1 - adjVal2)
	}

	return errSum / float64(card)
}

// HellingerDiff returns the Hellinger error between the model's current
// marginal estimate and this solution. Like AbsError, the result is the
// average over the variables, the solution's marginals are assumed normalized
// (sum=1.0), while the model's marginals are assumed non-normalized (but
// positive)
func HellingerDiff(v1 *Variable, v2 *Variable) float64 {
	card := v1.Card

	// get totals for normalizing
	tot1, tot2 := float64(0.0), float64(0.0)
	const eps = 1e-12

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

	// Hellinger distance is similar to the Euclidean L2:
	// sum((sqrt(p) - sqrt(q))**2) / sqrt(2)
	errSum := float64(0.0)
	for c := 0; c < card; c++ {
		adjVal1 := math.Sqrt(v1.Marginal[c] / tot1)
		adjVal2 := math.Sqrt(v2.Marginal[c] / tot2)
		err := math.Pow(adjVal1-adjVal2, 2) // squared, so always positive
		errSum += err
	}
	return errSum / math.Sqrt2
}

// klDivergence returns the Kullback–Leibler divergence, which is
// non-symmetric! This is strictly a subroutine for JS Divergence, so there
// is no error checking, the marginal values are operated on directly, and
// the arrays are assumed normalized (so sum(p1) == sum(p2) == 1.0)
// klDivergence(P, Q) <==> D_{KL}(P || Q)
func klDivergence(v1 []float64, v2 []float64) float64 {
	diverge := float64(0.0)
	for i, p1 := range v1 {
		p2 := v2[i]
		diverge += p1 * math.Log2(p1/p2)
	}

	return diverge
}

// JSDivergence returns the Jensen-Shannon divergence, which is a
// symmetric gneralization of the KL divergence
func JSDivergence(v1 *Variable, v2 *Variable) float64 {
	const eps = float64(1e-12)

	card := v1.Card

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

	p1Norm := make([]float64, card)
	p2Norm := make([]float64, card)
	mid := make([]float64, card)
	for i, p1 := range v1.Marginal {
		p2 := v2.Marginal[i]
		p1Norm[i] = p1 / tot1
		p2Norm[i] = p2 / tot2
		mid[i] = (p1Norm[i] + p2Norm[i]) * 0.5
	}

	return 0.5 * (klDivergence(p1Norm, mid) + klDivergence(p2Norm, mid))
}
