package sampler

import (
	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/pkg/errors"
)

// varSet is a set of variables, used to track the neighborhood for a variable
type varSet map[int]bool

// GibbsCollapsed supports collapsing specified variables
// It is a smart wrapper around our gibbs-simple sampler.
type GibbsCollapsed struct {
	baseSampler  *GibbsSimple
	varNeighbors []varSet
}

// NewGibbsCollapsed creates a new sampler
func NewGibbsCollapsed(gen *rand.Generator, m *model.Model) (*GibbsCollapsed, error) {
	base, err := NewGibbsSimple(gen, m)
	if base == nil {
		return nil, errors.Wrap(err, "Base simple Gibbs sampler could not be created")
	}

	// A lookup from variables to their neighbors
	neighbors := make([]varSet, len(base.pgm.Vars))

	// Create a neighbor entry per variable
	for i, v := range base.pgm.Vars {
		if i != v.ID {
			return nil, errors.Errorf("Invalid variable setup: [%d] => %+v", i, v)
		}
		neighbors[i] = make(varSet)
	}

	// Use the Gibbs Simple varFuncs lookup to find all connected variables
	for idx, funcs := range base.varFuncs {
		for _, f := range funcs {
			for _, v := range f.Vars {
				neighbors[idx][v.ID] = true
			}
		}
	}

	s := &GibbsCollapsed{
		baseSampler:  base,
		varNeighbors: neighbors,
	}

	return s, nil
}

// NeighborVarMax is the max size of the neighborhood allowed for a
// variable that we will collapse. Note that it includes the variable itself,
// so the total size of input space is 2^(M-1) where M is NeighborVarMax.
const NeighborVarMax = 22

// Collapse integrates out the variable given by index. If the index is < 0, a
// variable is randomly chosen. The collapsed variable is returned for
// inspection.
func (g *GibbsCollapsed) Collapse(varIdx int) (*model.Variable, error) {
	base := g.baseSampler
	pgm := base.pgm

	if varIdx < 0 {
		// Select random variable that is not collapsed and not fixed, but
		// we only select variables that are tractable - and we only try
		// N times (where N is our variable count)
		var err error
		for i := 0; i < len(pgm.Vars); i++ {
			varIdx, err = base.varSelector.VarSample(pgm.Vars, true)
			if err != nil {
				return nil, errors.Wrapf(err, "Failure selecting random variable to collapse")
			}

			nCount := len(g.varNeighbors[varIdx])
			if nCount <= NeighborVarMax {
				break
			} else {
				varIdx = -1
			}
		}
	}

	if varIdx < 0 {
		return nil, errors.Errorf("Failed to randomly select a variable to collapse")
	}

	if varIdx >= len(pgm.Vars) {
		return nil, errors.Errorf("Invalid variable index: max is %d", len(pgm.Vars)-1)
	}

	v := g.baseSampler.pgm.Vars[varIdx]
	if v.FixedVal >= 0 {
		return nil, errors.Errorf("Can not collapsed Fixed Val variable %v:%v", v.ID, v.Name)
	}

	// TODO: alert on intractable variable (too many source variables in
	//       factors to collapse. (This should factor in with our random
	//       selection above). We are going to initially limit the Markov
	//       blanket to 21 variables (so space is roughly 2^21)
	// TODO: actually collapse variable
	v.Collapsed = true

	return v, nil
}

// Sample returns a single sample - implements FullSampler
func (g *GibbsCollapsed) Sample(s []int) (int, error) {
	pgm := g.baseSampler.pgm

	if len(s) != len(pgm.Vars) {
		return -1, errors.Errorf("Samples size %d is wrong", len(s))
	}

	// Note excludeCollapsed=True
	varIdx, err := g.baseSampler.varSelector.VarSample(pgm.Vars, true)
	if err != nil {
		return -1, err
	}

	v := pgm.Vars[varIdx]
	if v.Collapsed {
		return -1, errors.Errorf(
			"Variable sampler selected collapsed variable %v:%v as index %v",
			v.ID, v.Name, varIdx,
		)
	}

	// Now we can just use the simple gibbs sampler
	return g.baseSampler.SampleVar(varIdx, s)
}
