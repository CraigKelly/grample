package sampler

import (
	"math/rand"

	"github.com/CraigKelly/grample/model"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/stat/sampleuv"
)

// GibbsSimple is our baseline, simple to code Gibbs sampler
type GibbsSimple struct {
	src         rand.Source
	rnd         *rand.Rand
	pgm         *model.Model
	varSelector sampleuv.Sampler
	last        []float64
}

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(src rand.Source, m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	s := &GibbsSimple{
		src:  src,
		rnd:  rand.New(src),
		pgm:  m,
		last: make([]float64, len(m.Vars)),
	}

	// Starting point in the sample space - note that the next call to Sample
	// will return the next sample, and not this one so our user will never
	// see this starting point unless they explicitly look for it.
	for i, v := range s.pgm.Vars {
		// Select value for every variable with uniform prob
		s.last[i] = float64(s.rnd.Int31n(int32(v.Card)))
	}

	return s, nil
}

// Sample returns a single sample
func (g *GibbsSimple) Sample(s []float64) error {
	if len(s) != len(g.pgm.Vars) {
		return errors.Errorf("Sample size %d != Var size %d in model %s", len(s), len(g.pgm.Vars), g.pgm.Name)
	}

	varIdx := 0    // TODO: select variable with g.varSelector
	nextVal := 0.0 // TODO: Create our next sample with g.varSamplers[varIdx]

	// Update saved copy with new value and copy to caller's sample
	g.last[varIdx] = nextVal
	copy(s, g.last)

	return nil
}
