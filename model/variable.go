package model

import (
	"math"

	"github.com/pkg/errors"
)

// Variable represents a single node in a PGM, a random variable, or a marginal distribution.
type Variable struct {
	Name     string    // Variable name (just a zero-based index in UAI formats)
	Card     int       // Cardinality - values are assume to be 0 to Card-1
	Marginal []float64 // Current best estimate for marginal distribution: len should equal Card
}

// Check returns an error if any problem is found
func (v *Variable) Check() error {
	if v.Card != len(v.Marginal) {
		return errors.Errorf("Variable %s Card %d != len(M) %d", v.Name, v.Card, len(v.Marginal))
	}

	// marginal should be a probability dist
	if v.Card > 0 {
		var sum float64
		for _, p := range v.Marginal {
			sum += p
		}

		const EPS = 1e-8
		if math.Abs(sum-1.0) >= EPS {
			return errors.Errorf("Variable %s has marginal dist with sum=%f", sum)
		}
	}

	return nil
}

// NormMarginal insures scales the current Marginal vector to sum to 1
func (v *Variable) NormMarginal() error {
	if v.Card != len(v.Marginal) {
		return errors.Errorf("Var %s - can not norm: Card=%d, Len(m)=%d", v.Card, len(v.Marginal))
	}

	if v.Card < 1 {
		return nil // Nothing to do
	}

	if v.Card == 1 {
		v.Marginal[0] = 1.0 // easy
	}

	// From here down, we actually need to do some work
	var sum float64
	for _, p := range v.Marginal {
		sum += p
	}

	const EPS = 1e-8

	if math.Abs(sum-1.0) < EPS {
		return nil // Already norm'ed
	}

	for i, p := range v.Marginal {
		v.Marginal[i] = p / sum
	}

	return nil
}
