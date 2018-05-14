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
	Funcs []*Function // Function of variables (CPT) in the model
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

// Solution to a marginal estimation problem specified on a Model
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

// Score returns the current value of the given evaluation metric. While the
// solution marginal is assumed to be normalized, the model variables probably
// will NOT be normalized.
func (s *Solution) Score(m *Model) (float64, error) {
	if len(s.Vars) != len(m.Vars) {
		return 0.0, errors.Errorf("Solution var count %d != model var count %d", len(s.Vars), len(m.Vars))
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

		// accumulate error (normalizing model var)
		for c := 0; c < v.Card; c++ {
			modelVal := v.Marginal[c] / tot
			totErr += math.Abs(modelVal - s.Vars[i].Marginal[c])
		}
	}

	return totErr / float64(len(s.Vars)), nil
}
