package model

import (
	"math"

	"github.com/pkg/errors"
)

// Variable represents a single node in a PGM, a random variable, or a marginal distribution.
type Variable struct {
	ID        int                // A numeric ID for tracking a variable
	Name      string             // Variable name (just a zero-based index in UAI formats)
	Card      int                // Cardinality - values are assume to be 0 to Card-1
	FixedVal  int                // Current fixed value (fixed by evidence): -1 is no evidence, else if 0 to Card-1
	Marginal  []float64          // Current best estimate for marginal distribution: len should equal Card
	State     map[string]float64 // State/stats a sampler can track - mainly for JSON tracking
	Collapsed bool               // For Collapsed == True, you should just sample from Marginal (default is False)
}

// NewVariable is our standard way to create a variable from an index and a
// cardinality. The marginal will be set to uniform.
func NewVariable(index int, card int) (*Variable, error) {
	if index < 0 {
		return nil, errors.Errorf("Invalid index %d with card %d", index, card)
	}
	if card < 1 {
		return nil, errors.Errorf("Invalid card %d for variable %d", card, index)
	}

	v := &Variable{
		ID:       index,
		Name:     "",
		Card:     card,
		FixedVal: -1,
		Marginal: make([]float64, card),
		State:    make(map[string]float64),
	}

	var err error
	err = v.CreateName(index)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not init name for var %d (card %d)", index, card)
	}

	err = v.NormMarginal()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not init norm marginal for var %d (card %d)", index, card)
	}

	return v, nil
}

// Clone returns a deep copy of the variable. Marginal is normalize, and the
// state dict is copied.
func (v *Variable) Clone() *Variable {
	cp := &Variable{
		ID:        v.ID,
		Name:      v.Name,
		Card:      v.Card,
		FixedVal:  v.FixedVal,
		Marginal:  make([]float64, v.Card),
		State:     make(map[string]float64),
		Collapsed: v.Collapsed,
	}

	for ky, val := range v.State {
		cp.State[ky] = val
	}

	copy(cp.Marginal, v.Marginal)

	return cp
}

// Check returns an error if any problem is found
func (v *Variable) Check() error {
	if v.Card != len(v.Marginal) {
		return errors.Errorf("Variable %s Card %d != len(M) %d", v.Name, v.Card, len(v.Marginal))
	}

	// FixedVal should be -1 or correspond to card
	// Note that this means you can never have a fixed value for a var with card 0.
	if v.FixedVal != -1 {
		if v.FixedVal < 0 || v.FixedVal >= v.Card {
			return errors.Errorf("Variable %s has fixed val %d but must be -1 or match card %d", v.Name, v.FixedVal, v.Card)
		}
	}

	// marginal should be a probability dist
	if v.Card > 0 {
		var sum float64
		for _, p := range v.Marginal {
			sum += p
		}

		const EPS = 1e-8
		if math.Abs(sum-1.0) >= EPS {
			return errors.Errorf("Variable %s has marginal dist with sum=%f", v.Name, sum)
		}
	}

	return nil
}

// NormMarginal insures/scales the current Marginal vector to sum to 1
func (v *Variable) NormMarginal() error {
	if v.Card != len(v.Marginal) {
		return errors.Errorf("Var %s - can not norm: Card=%d, Len(m)=%d", v.Name, v.Card, len(v.Marginal))
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

	// Can stop if already normed
	if math.Abs(sum-1.0) < EPS {
		return nil
	}

	// If sum is 0, we just assume uniformity
	if math.Abs(sum) < EPS {
		p := 1.0 / float64(v.Card)
		for i := range v.Marginal {
			v.Marginal[i] = p
		}
		return nil
	}

	// norm
	for i, p := range v.Marginal {
		v.Marginal[i] = p / sum
	}

	return nil
}

// CreateName just gives a name to variable based on a numeric index
func (v *Variable) CreateName(i int) error {
	if i < 0 {
		return errors.Errorf("Invalid index %d for CreateName - must be >= 0", i)
	}

	// Just use a letter scheme (similar to Excel columns)
	v.Name = letter26(i)
	return nil
}

func divmod(numerator, denominator int) (quotient, remainder int) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder = numerator % denominator
	return
}

// letter26 is sort of base-26 with only letters, but A=0 *and* the start digit (so 0=A, 1=B, and ZZ+1=AAA)
func letter26(n int) string {
	// Easy for n==0
	if n == 0 {
		return "A"
	}
	// Need to bump up one
	n++

	const LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := make([]byte, 0, 8)
	var remain int
	for n > 0 {
		n, remain = divmod(n-1, 26)
		digits = append(digits, LETTERS[remain])
	}

	//reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}
