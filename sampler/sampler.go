package sampler

import (
	"sync"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/pkg/errors"
)

// A FullSampler populates the given array with values from the model (e.e.
// Gibbs sampling).  The array MUST be the same size as the variables being
// sampled. Note that this call pattern (mutable param) parallels the samplers
// in the gonum stats subpackags. Samplers update the int slice and then return
// the index of the selected variable updated.
type FullSampler interface {
	Sample([]int) (int, error)
}

// An AdaptiveSampler accepts a list of current chains and returns a new list
// ready to advance. The simplest AdaptiveSampler just returns the chains
// passed and is equivalent to whatever base sampler is currently in use.
type AdaptiveSampler interface {
	Adapt(chains []*Chain, newChainCount int) ([]*Chain, error)
}

// A VarSampler selects from an array of variables with some probability.
// Currently used yo select the next variable to sample in a chain in our Gibbs
// sampler. The selection routine should exclude variables with a Fixed Value
// and optionally exclude Variable with Collapsed==true.
type VarSampler interface {
	VarSample(vs []*model.Variable, excludeCollapsed bool) (int, error)
}

// WeightedSampler provides a way to select from a cardinality of values given
// an array of weights.
type WeightedSampler interface {
	WeightedSample(card int, weights []float64) (int, error)
}

// UniformSampler provides uniform sampling for our interfaces
type UniformSampler struct {
	gen  *rand.Generator
	pool *sync.Pool
}

// NewUniformSampler creates a new uniform sampler
func NewUniformSampler(gen *rand.Generator, maxVars int) (*UniformSampler, error) {
	if maxVars < 1 {
		return nil, errors.Errorf("Invalid max var count (%d)", maxVars)
	}

	p := &sync.Pool{
		New: func() interface{} {
			return make([]int, maxVars)
		},
	}

	s := &UniformSampler{
		gen:  gen,
		pool: p,
	}
	return s, nil
}

// maxCard is the maximum cardinality supported by our UniSample and
// WeightedSample below.
const maxCard = 1 << 30

// UniSample samples uniformly from [0, card).
func (s *UniformSampler) UniSample(card int) (int, error) {
	if card < 1 {
		return -1, errors.New("Can not sample if Cardinality < 1")
	}

	if card > maxCard {
		return -1, errors.Errorf("Cardinality above %d not supported", maxCard)
	}

	if card == 1 {
		return 0, nil
	}

	return int(s.gen.Int31n(int32(card))), nil
}

// WeightedSample samples from [0, card) based on the card-sized array of
// weights. Mainly for sampling directly from a variable's marginal.
func (s *UniformSampler) WeightedSample(card int, weights []float64) (int, error) {
	if card < 1 {
		return -1, errors.New("Can not sample if Cardinality < 1")
	}

	if card > maxCard {
		return -1, errors.Errorf("Cardinality above %d not supported", maxCard)
	}

	if len(weights) != card {
		return -1, errors.Errorf("Weight array size %d must match cardinality %d", len(weights), card)
	}

	if card == 1 {
		return 0, nil
	}

	totWeight := 0.0
	for _, w := range weights {
		if w <= 0.0 {
			return -1, errors.Errorf("Weights must be > 0.0")
		}
		totWeight += w
	}

	r := s.gen.Float64() * totWeight
	selVal := -1
	for i, w := range weights {
		if r <= w {
			selVal = i
			break
		}
		r -= w
	}

	if selVal < 0 {
		return -1, errors.Errorf("Failed to sample for card  %v (%+v)", card, weights)
	}

	return selVal, nil
}

// VarSample implements VarSample interface. If excludeCollapsed is true, no
// collapsed variable will be selected. Variable with a Fixed Val will never be
// selected.
func (s *UniformSampler) VarSample(vs []*model.Variable, excludeCollapsed bool) (int, error) {
	vsLen := len(vs)
	if vsLen < 1 {
		return 0, errors.New("Can not sample from an empty variable list")
	}

	// First find indexes of all variables we can select (that are NOT fixed)
	targetIndexes := s.pool.Get().([]int)
	defer s.pool.Put(targetIndexes)

	targetCount := 0
	for i, v := range vs {
		if excludeCollapsed && v.Collapsed {
			continue
		}
		if v.FixedVal >= 0 {
			continue
		}

		targetIndexes[targetCount] = i
		targetCount++
	}

	// Corner cases
	if targetCount < 1 {
		// No possible selection
		return 0, errors.New("No Variables to select")
	} else if targetCount == 1 {
		// Only one variable to select
		return targetIndexes[0], nil
	}

	// Select an entry from our list and return the corresponding index
	i, e := s.UniSample(targetCount)
	if e != nil {
		return -1, e
	}

	return targetIndexes[i], nil
}
