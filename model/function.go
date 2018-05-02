package model

import (
	"github.com/pkg/errors"
)

// Function represents a function of Variables or a CPT.
// TODO: describe order of Table
type Function struct {
	Name  string      // Name for function (or just a 0-based index in UAI formats)
	Vars  []*Variable // Vars in function
	Table []float64   // CPT - len is product of variables' Card
}

// Check returns an error if any problem is found
func (f Function) Check() error {
	expTableSize := 0

	if len(f.Vars) > 0 {
		expTableSize = 1

		for _, v := range f.Vars {
			if v.Card < 1 {
				return errors.Errorf("Variable %s has card %d but is in Function %s", v.Name, v.Card, f.Name)
			}
			expTableSize *= v.Card
		}
	}

	if expTableSize != len(f.Table) {
		return errors.Errorf("Function %s expected table size %d, found %d", f.Name, expTableSize, len(f.Table))
	}

	// TODO: ensure Table is valid (sums to 1.0 for in all cases)?

	return nil
}
