package model

import (
	"github.com/pkg/errors"
)

// VariableIter is an iterator over all possible values for a list of variables
type VariableIter struct {
	vars    []*Variable
	lastVal []int
}

// NewVariableIter returns a new iterator over the list of variables
func NewVariableIter(src []*Variable) (*VariableIter, error) {
	if len(src) < 1 {
		return nil, errors.Errorf("At least one variable required for iteration")
	}

	vi := &VariableIter{
		vars:    make([]*Variable, len(src)),
		lastVal: make([]int, len(src)),
	}

	copy(vi.vars, src) // Note: we don't clone

	return vi, nil
}

// Val populates curr with the current value
func (vi *VariableIter) Val(curr []int) error {
	if len(curr) < len(vi.lastVal) {
		return errors.Errorf("Dest buffer of size %d needs to be %d", len(curr), len(vi.lastVal))
	}

	copy(curr, vi.lastVal)

	return nil
}

// Next advances to the next value and returns True if there are still values to see
func (vi *VariableIter) Next() bool {
	for i := len(vi.vars) - 1; i >= 0; i-- {
		prop := vi.lastVal[i] + 1

		if prop < vi.vars[i].Card {
			// All done
			vi.lastVal[i] = prop
			return true
		}

		vi.lastVal[i] = 0 // Overflow: continue to next
	}

	// If we're still here then we set every digit to 0 and wrapped around
	return false
}
