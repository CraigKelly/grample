package model

import (
	"github.com/pkg/errors"
)

// VariableIter is an iterator over all possible values for a list of variables
type VariableIter struct {
	vars       []*Variable
	lastVal    []int
	honorFixed bool // If true, vars with a FixedVal always take that value
}

// NewVariableIter returns a new iterator over the list of variables
func NewVariableIter(src []*Variable, honorFixed bool) (*VariableIter, error) {
	if len(src) < 1 {
		return nil, errors.Errorf("At least one variable required for iteration")
	}

	vi := &VariableIter{
		vars:       make([]*Variable, len(src)),
		lastVal:    make([]int, len(src)),
		honorFixed: honorFixed,
	}

	copy(vi.vars, src) // Note: we don't clone

	// Set initial value to include Fixed Vals if that's what they want
	if vi.honorFixed {
		for i, v := range vi.vars {
			if v.FixedVal >= 0 {
				vi.lastVal[i] = v.FixedVal
			}
		}
	}

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
		v := vi.vars[i]
		// Special case: fixed values never change (if honorFixed=True)
		if vi.honorFixed && v.FixedVal >= 0 {
			vi.lastVal[i] = v.FixedVal
			continue
		}

		prop := vi.lastVal[i] + 1

		if prop < v.Card {
			// All done
			vi.lastVal[i] = prop
			return true
		}

		vi.lastVal[i] = 0 // Overflow: continue to next
	}

	// If we're still here then we set every digit to 0 and wrapped around
	return false
}
