package model

import (
	"io/ioutil"
	"math"
	"path/filepath"

	"github.com/pkg/errors"
)

// Model type constant string - matches UAI formats
const (
	BAYES  = "BAYES"
	MARKOV = "MARKOV"
)

// Reader implementors instantiate a model from a byte stream
type Reader interface {
	ReadModel(data []byte) (*Model, error)
}

// SolReader implementors read a solution (currently we only support marginal solutions)
type SolReader interface {
	ReadMargSolution(data []byte) (*Solution, error)
}

// TODO: Evidence reader

// Model represent a PGM
type Model struct {
	Type  string      // PGM type - should match a constant
	Name  string      // Model name
	Vars  []*Variable // Variables (nodes) in the model
	Funcs []*Function `json:"-"` // Function of variables (CPT) in the model
}

// NewModelFromFile initializes and creates a model from the specified source.
func NewModelFromFile(r Reader, filename string) (*Model, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not READ model from %s", filename)
	}

	model, err := NewModelFromBuffer(r, data)
	if err != nil {
		return nil, err
	}

	// Name the model from the file
	var ext = filepath.Ext(filename)
	model.Name = filename[0 : len(filename)-len(ext)]

	return model, nil
}

// NewModelFromBuffer creates a model from the given pre-read data
func NewModelFromBuffer(r Reader, data []byte) (*Model, error) {
	m, err := r.ReadModel(data)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not PARSE model")
	}

	err = m.Check()
	if err != nil {
		return nil, errors.Wrapf(err, "Parsed model is not valid")
	}

	return m, nil
}

// Check returns an error if there is a problem with the model
func (m *Model) Check() error {
	if m.Type != BAYES && m.Type != MARKOV {
		return errors.Errorf("Unknown model type %s", m.Type)
	}

	varID := make(map[int]bool)
	for _, v := range m.Vars {
		e := v.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Variable %s", m.Name, v.Name)
		}

		_, ok := varID[v.ID]
		if ok {
			return errors.Errorf("Duplicate Id %d for Var %s", v.ID, v.Name)
		}
		varID[v.ID] = true
	}

	for _, f := range m.Funcs {
		e := f.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Function %s", m.Name, f.Name)
		}
	}

	return nil
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

	for i, v := range m.Vars {
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

	absErrMean = totErrSum / float64(len(s.Vars))
	maxErrMean = totErrMax / float64(len(s.Vars))
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

	for i, v := range m.Vars {
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

	return totErr / float64(len(s.Vars)), nil
}
