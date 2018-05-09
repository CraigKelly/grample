package sampler

import (
	"math/rand"

	"github.com/CraigKelly/grample/model"
	"github.com/pkg/errors"
)

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
	src rand.Source
	rnd *rand.Rand
}

// NewUniformSampler creates a new uniform sampler
func NewUniformSampler(src rand.Source) (*UniformSampler, error) {
	s := &UniformSampler{
		src: src,
		rnd: rand.New(src),
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

	return int(s.rnd.Int31n(int32(card))), nil
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
