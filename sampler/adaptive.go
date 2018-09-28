package sampler

import (
	"github.com/CraigKelly/grample/model"
	//"github.com/pkg/errors"
)

// TODO: unit testing

// IdentitySampler is just a non-adaptive strategy.
type IdentitySampler struct{}

// NewIdentitySampler create a new IdentitySampler
func NewIdentitySampler() (*IdentitySampler, error) {
	return &IdentitySampler{}, nil
}

// Adapt for an IdentitySampler is just an identity operation (thus the clever
// name :)
func (i *IdentitySampler) Adapt(chains []*Chain) ([]*Chain, error) {
	return chains, nil
}

// ConvergenceSampler creates new collapsed chains based on convergence
// metrics.
type ConvergenceSampler struct {
	BaseModel     *model.Model
	NewChainCount int
	MaxChains     int
}

// NewConvergenceSampler create a new IdentitySampler
func NewConvergenceSampler(m *model.Model) (*ConvergenceSampler, error) {
	// TODO: chain count and max chains should be parameterized AND on the
	//       command line
	s := &ConvergenceSampler{
		BaseModel:     m,
		NewChainCount: 2,
		MaxChains:     128,
	}
	return s, nil
}

// Adapt for a ConvergenceSampler creates new chains with collapsed variables.
// The variable to collapse is selected from a probability distribution
// weighted by a convergence metric.
func (c *ConvergenceSampler) Adapt(chains []*Chain) ([]*Chain, error) {
	// TODO: get convergence
	// TODO: all non-collapsed, non-fixed variables sorted by convergence metric
	// TODO: select from our list
	// TODO: handle len(chains) > c.MaxChains
	return chains, nil
}
