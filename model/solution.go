package model

import (
	"io/ioutil"
	"math"

	"github.com/pkg/errors"
)

// SolReader implementors read a solution (currently we only support marginal solutions)
type SolReader interface {
	ReadMargSolution(data []byte) (*Solution, error)
}

// Solution to a marginal estimation problem specified on a Model. It also
// provides evaluation metrics to evaluate vs the solution.
type Solution struct {
	Vars []*Variable // Variables with their marginals
}

// NewSolutionFromFile reads a UAI MAR solution file
func NewSolutionFromFile(r SolReader, filename string) (*Solution, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not READ solution from %s", filename)
	}

	sol, err := NewSolutionFromBuffer(r, data)
	if err != nil {
		return nil, err
	}

	return sol, nil
}

// NewSolutionFromBuffer reads a UAI MAR solution file from the specified buffer
func NewSolutionFromBuffer(r SolReader, data []byte) (*Solution, error) {
	s, err := r.ReadMargSolution(data)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not PARSE solution")
	}

	return s, nil
}

// Check insures that the solution is as correct as can be checked given a model
func (s *Solution) Check(m *Model) error {
	for _, v := range s.Vars {
		e := v.Check()
		if e != nil {
			return errors.Wrapf(e, "Solution has an invalid Variable %s", v.Name)
		}
	}

	if len(s.Vars) != len(m.Vars) {
		return errors.Errorf("Solution var count %d != model var count %d", len(s.Vars), len(m.Vars))
	}

	return nil
}

// AbsError returns both the total and max absolute error between the model's
// current marginal estimations and this solution. The final score is the mean
// over all variables. The solution marginal is assumed to be normalized, the
// model variables are NOT.
func (s *Solution) AbsError(m *Model) (absErrMean float64, maxErrMean float64, failed error) {
	if len(s.Vars) != len(m.Vars) {
		return math.NaN(), math.NaN(), errors.Errorf("Solution var count %d != model var count %d", len(s.Vars), len(m.Vars))
	}

	totErrSum := float64(0.0) // Total error
	totErrMax := float64(0.0) // Total MAX error (max per var)
	const eps = float64(1e-12)

	// Remember that we skip vars
	varCount := 0

	for i, v := range m.Vars {
		if v.FixedVal >= 0 {
			continue
		}

		varCount++

		// get total for normalizing
		tot := float64(0.0)
		for c := 0; c < v.Card; c++ {
			tot += v.Marginal[c]
		}
		if tot < eps {
			tot = eps
		}

		// accumulate error (normalizing model var)
		maxErr := float64(0.0)
		for c := 0; c < v.Card; c++ {
			modelVal := v.Marginal[c] / tot
			err := math.Abs(modelVal - s.Vars[i].Marginal[c])

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
func (s *Solution) HellingerError(m *Model) (float64, error) {
	if len(s.Vars) != len(m.Vars) {
		return math.NaN(), errors.Errorf("Solution var count %d != model var count %d", len(s.Vars), len(m.Vars))
	}

	totErr := float64(0.0)
	const eps = float64(1e-12)

	// No fixed vars
	varCount := 0

	for i, v := range m.Vars {
		if v.FixedVal >= 0 {
			continue
		}

		varCount++

		// get total for normalizing
		tot := float64(0.0)
		for c := 0; c < v.Card; c++ {
			tot += v.Marginal[c]
		}
		if tot < eps {
			tot = eps
		}

		// accumulate error (normalizing model var). Hellinger distance is
		// similar to the Euclidean L2: sum((sqrt(p) - sqrt(q))**2) / sqrt(2)
		errSum := float64(0.0)
		for c := 0; c < v.Card; c++ {
			modelVal := math.Sqrt(v.Marginal[c] / tot)
			solVal := math.Sqrt(s.Vars[i].Marginal[c])
			err := math.Pow(modelVal-solVal, 2)
			errSum += err
		}
		totErr += errSum / math.Sqrt2
	}

	if varCount < 1 {
		return math.NaN(), errors.Errorf("No un-fixed vars to score")
	}

	return totErr / float64(varCount), nil
}
