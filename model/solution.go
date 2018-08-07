package model

import (
	"io/ioutil"

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

// Error is a helper method to return the entire error suite we offer for the
// current solution against the given model
func (s *Solution) Error(m *Model) (*ErrorSuite, error) {
	return NewErrorSuite(s.Vars, m.Vars)
}
