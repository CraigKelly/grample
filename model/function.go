package model

import (
	"math"

	"github.com/pkg/errors"
)

// A Function represents a function of Variables (which may be a CPT or a more
// general factor). In a Markov network, this is defined on a clique. Note that
// factors in a Markox network are assumed to return NON-normalized
// probabilities.  You need Z (the partition function) to normalize to "real"
// probabilities.
//
// The actual ordering of the Table values matches the order of the variables,
// where the variables are ordered from "most" to "least" significant. (This is
// the same order used in UAI data files). As a example, let's assume 3 boolean
// variables in the order [A, B, C]. Let's further assume that this join
// probability distribution is completely uniform: The CPT would look like:
//
//   ABC  P(A,B,C)
//   ---  --------
//   000   0.125
//   001   0.125
//   010   0.125
//   011   0.125
//   100   0.125
//   101   0.125
//   110   0.125
//   111   0.125
//
// And our Table array would be in the same order. Since we assume that a
// variable's domain is [0, C-1] where C is cardinality (e.g. a boolean var has
// C=2 with values {0,1}), we can map directory from an ordered list of values
// (in the same order as the variables) to an index in the table (see Eval).
type Function struct {
	Name  string      // Name for function (or just a 0-based index in UAI formats)
	Vars  []*Variable // Vars in function
	Table []float64   // CPT - len is product of variables' Card
}

// Check returns an error if any problem is found
func (f *Function) Check() error {
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

	return nil
}

// Eval returns the result of the function
func (f *Function) Eval(values []int) (float64, error) {
	i, err := f.calcIndex(values)
	if err != nil {
		return math.NaN(), err
	}

	if i < 0 || i >= len(f.Table) {
		return math.NaN(), errors.Errorf("Could not find table entry for values %v", values)
	}

	return f.Table[i], nil
}

// calcIndex generates an index into the table given a vector of values.
func (f *Function) calcIndex(values []int) (int, error) {
	if len(values) != len(f.Vars) {
		return -1, errors.Errorf("Value vector %v does not match variables", values)
	}

	// Work from least significant to most significant. (This is not optional:
	// each "digit" can be a different size).
	digit := 1
	location := 0

	for i := len(values) - 1; i >= 0; i-- {
		val := values[i]
		card := f.Vars[i].Card
		if val < 0 || val >= card {
			return -1, errors.Errorf("Value %d invalid for cardinality %d for var %s", val, card, f.Vars[i].Name)
		}

		location += digit * val
		digit *= card
	}

	return location, nil
}
