package sampler

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
)

// IdentitySampler is just a non-adaptive strategy.
type IdentitySampler struct{}

// NewIdentitySampler create a new IdentitySampler
func NewIdentitySampler() (*IdentitySampler, error) {
	return &IdentitySampler{}, nil
}

// Adapt for an IdentitySampler is just an identity operation (thus the clever
// name :)
func (i *IdentitySampler) Adapt(chains []*Chain, newChainCount int) ([]*Chain, error) {
	return chains, nil
}

// ConvergenceSampler creates new collapsed chains based on convergence
// metrics.
type ConvergenceSampler struct {
	BaseModel *model.Model
	DistFunc  Measure
	Gen       *rand.Generator
	MaxChains int
}

// NewConvergenceSampler create a new IdentitySampler.
func NewConvergenceSampler(gen *rand.Generator, m *model.Model, d Measure) (*ConvergenceSampler, error) {
	if m == nil {
		return nil, errors.Errorf("A full model is required for Adaptation")
	}

	if d == nil {
		d = model.HellingerDiff
	}

	s := &ConvergenceSampler{
		BaseModel: m,
		DistFunc:  d,
		Gen:       gen,
		MaxChains: 128,
	}
	return s, nil
}

// Adapt for a ConvergenceSampler creates new chains with collapsed variables.
// The variable to collapse is selected from a probability distribution
// weighted by a convergence metric.
func (c *ConvergenceSampler) Adapt(chains []*Chain, newChainCount int) ([]*Chain, error) {
	if len(chains) < 2 {
		return nil, errors.Errorf("At least 2 chains required for adaptation")
	}

	if len(chains) >= c.MaxChains {
		return chains, nil
	}

	// Go ahead and create the collapsed sampler we'll need - note this gets us
	// blanket sizes as well.
	modClone := c.BaseModel.Clone()
	samp, err := NewGibbsCollapsed(c.Gen, modClone)
	if err != nil {
		return nil, err
	}

	// Build an array of non-collapsed, non-fixed variables that have a
	// resonable-sized neighborhood. To do this we need to merge the chains
	mergedVars, err := MergeChains(chains)
	if err != nil {
		return nil, err
	}

	vars := make([]*model.Variable, 0, len(mergedVars))
	for _, v := range mergedVars {
		if v.FixedVal < 0 && !v.Collapsed && samp.BlanketSize(v) <= NeighborVarMax {
			vars = append(vars, v)
		}
	}

	// Nothing left to do
	if len(vars) < 1 {
		return chains, nil
	}

	targetVarIdxs := make([]int, 0, newChainCount)
	if len(vars) <= newChainCount {
		for _, v := range vars {
			targetVarIdxs = append(targetVarIdxs, v.ID)
		}
	} else {
		// Get convergence for our variables - note that we have already merged
		// variables, so we can use those
		converge, err := ChainConvergence(chains, c.DistFunc, mergedVars)
		if err != nil {
			return nil, err
		}

		// Sort by convergence diagnostic and choose var with highest score
		// (Worst convergence).  IMPORTANT: we are sorting instead of just
		// scanning because eventually we'll want to select stochastically from
		// a dist weighted by score
		sort.Slice(vars, func(i, j int) bool {
			return converge[vars[i].ID] > converge[vars[j].ID]
		})

		pos := len(vars) - 1
		for cc := 0; cc < newChainCount; cc++ {
			targetVarIdxs = append(targetVarIdxs, vars[pos].ID)
			pos--
		}
	}

	if len(targetVarIdxs) < 1 {
		return chains, nil
	}

	lastChain := chains[len(chains)-1]

	// Note that we our sampler from above and then nil it: this is so chains
	// 2+ get their own sampler
	for _, varIdx := range targetVarIdxs {
		if samp == nil {
			modClone = c.BaseModel.Clone()
			samp, err = NewGibbsCollapsed(c.Gen, modClone)
			if err != nil {
				return nil, err
			}
		}

		// Now we know enough to collapse our variable and create a new chain
		_, err = samp.Collapse(varIdx)
		if err != nil {
			return nil, err
		}

		newChain, err := NewChain(modClone, samp, lastChain.ConvergenceWindow, 2)
		if err != nil {
			return nil, err
		}

		chains = append(chains, newChain)

		// Reset after we use a sampler
		samp = nil
	}

	return chains, nil
}
