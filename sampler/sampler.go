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
// in the gonum stats subpackags.
type FullSampler interface {
	Sample([]int) error
}

// A ValSampler returns a sample given a cardinality. We assume the possible
// values are 0 to Cardinality-1. Mainly used to select a starting point for a
// Gibbs-style sampler.
type ValSampler interface {
	ValSample(card int) (int, error)
}

// A VarSampler selects from an array of variables with some probability.
// Currently used select the next variable to sample in a chain in our Gibbs
// sampler.
type VarSampler interface {
	VarSample(vs []*model.Variable) (int, error)
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

// ValSample implements ValSampler interface
func (s *UniformSampler) ValSample(card int) (int, error) {
	if card < 1 {
		return -1, errors.New("Can not sample if Cardinality < 1")
	}
	if card == 1 {
		return 0, nil
	}

	return int(s.gen.Int31n(int32(card))), nil
}

// VarSample implements VarSample interface
func (s *UniformSampler) VarSample(vs []*model.Variable) (int, error) {
	vsLen := len(vs)
	if vsLen < 1 {
		return 0, errors.New("Can not sample from an empty variable list")
	}

	// First find indexes of all variables we can select (that are NOT fixed)
	targetIndexes := s.pool.Get().([]int)
	defer s.pool.Put(targetIndexes)

	targetCount := 0
	for i, v := range vs {
		if v.FixedVal < 0 {
			targetIndexes[targetCount] = i
			targetCount++
		}
	}

	// Corner cases
	if targetCount < 1 {
		// No possible selection
		return 0, errors.New("All variable are fixed - nothing to select")
	} else if targetCount == 1 {
		// Only one variable to select
		return targetIndexes[0], nil
	}

	// Select an entry from our list and return the corresponding index
	i, e := s.ValSample(targetCount)
	if e != nil {
		return -1, e
	}

	return targetIndexes[i], nil
}
