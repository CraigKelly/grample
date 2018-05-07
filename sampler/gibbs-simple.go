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
	varSamplers []sampleuv.Sampler
	last        []float64
}

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(src rand.Source, m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	s := &GibbsSimple{
		src: src,
		rnd: rand.New(src),
		pgm: m,
	}

	// TODO: create varSelector over variables (uniform)
	// TODO: create array for varSamplers - each is weighted for the var's Card
	// TODO: init var samplers

	return s, nil
}

// Sample returns a single sample
func (g *GibbsSimple) Sample(s []float64) error {
	if len(s) != len(g.pgm.Vars) {
		return errors.Errorf("Sample size %d != Var size %d in model %s", len(s), len(g.pgm.Vars), g.pgm.Name)
	}

	if len(g.last) < 1 {
		// Initial: we sample initial values at random for all variables
		g.last = make([]float64, len(g.pgm.Vars))
		for i, v := range g.pgm.Vars {
			// TODO: should we use the samplers?
			g.last[i] = float64(g.rnd.Int31n(int32(v.Card)))
		}
		copy(s, g.last)
		return nil
	}

	// TODO: update var samplers?

	varIdx := 0    // TODO: select variable with g.varSelector
	nextVal := 0.0 // TODO: Create our next sample with g.varSamplers[varIdx]

	// Update saved copy with new value and copy to caller's sample
	g.last[varIdx] = nextVal
	copy(s, g.last)

	return nil
}
