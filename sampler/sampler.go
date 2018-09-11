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

// A VarSampler selects from an array of variables with some probability.
// Currently used yo select the next variable to sample in a chain in our Gibbs
// sampler. The selection routine should exclude variables with a Fixed Value
// and optionally exclude Variable with Collapsed==true.
type VarSampler interface {
	VarSample(vs []*model.Variable, excludeCollapsed bool) (int, error)
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

// UniSample samples uniformly from [0, card).
func (s *UniformSampler) UniSample(card int) (int, error) {
	if card < 1 {
		return -1, errors.New("Can not sample if Cardinality < 1")
	}

	const maxCard = 1 << 30
	if card > maxCard {
		return -1, errors.Errorf("Cardinality above %d not supported", maxCard)
	}

	if card == 1 {
		return 0, nil
	}

	return int(s.gen.Int31n(int32(card))), nil
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
