package sampler

import (
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
// values are 0 to Cardinality-1.
type ValSampler interface {
	ValSample(card int) (int, error)
}

// A VarSampler selects from an array of variables with some probability
type VarSampler interface {
	VarSample(vs []*model.Variable) (int, error)
}

// UniformSampler provides uniform sampling for our interfaces
type UniformSampler struct {
	gen *rand.Generator
}

// NewUniformSampler creates a new uniform sampler
func NewUniformSampler(gen *rand.Generator) (*UniformSampler, error) {
	s := &UniformSampler{
		gen: gen,
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

	i, e := s.ValSample(vsLen)
	if e != nil {
		return -1, e
	}

	return i, nil
}
