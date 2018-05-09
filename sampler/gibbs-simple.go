package sampler

import (
	"math"
	"math/rand"

	"github.com/CraigKelly/grample/model"
	"github.com/pkg/errors"
)

// GibbsSimple is our baseline, simple to code Gibbs sampler
type GibbsSimple struct {
	src         rand.Source
	rnd         *rand.Rand
	pgm         *model.Model
	varSelector VarSampler
	last        []float64
}

// TODO: unit test error handling and getting at least one good sample

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(src rand.Source, m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	// Use log space for factors
	for _, f := range m.Funcs {
		err := f.UseLogSpace()
		if err != nil {
			return nil, errors.Wrapf(err, "Could not convert function %v to Log Space", f.Name)
		}
	}

	uniform, err := NewUniformSampler(src)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create uniform sampler in Gibbs Simple sample")
	}

	s := &GibbsSimple{
		src:         src,
		rnd:         rand.New(src),
		pgm:         m,
		varSelector: uniform,
		last:        make([]float64, len(m.Vars)),
	}

	// Starting point in the sample space - note that the next call to Sample
	// will return the next sample, and not this one so our user will never
	// see this starting point unless they explicitly look for it.
	for i, v := range s.pgm.Vars {
		val, err := uniform.ValSample(v.Card)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not generate start sample for variable %v", v.Name)
		}
		// Select value for every variable with uniform prob
		s.last[i] = float64(val)
	}

	return s, nil
}

// Sample returns a single sample
func (g *GibbsSimple) Sample(s []float64) error {
	if len(s) != len(g.pgm.Vars) {
		return errors.Errorf("Sample size %d != Var size %d in model %s", len(s), len(g.pgm.Vars), g.pgm.Name)
	}

	// Select next variable to sample
	varIdx, err := g.varSelector.VarSample(g.pgm.Vars)
	if err != nil {
		return errors.Wrapf(err, "Could not sample from vars in model %s", g.pgm.Name)
	}
	sampleVar := g.pgm.Vars[varIdx]

	// Find all related factors and marginalize for sampleVar
	sampleWeights := make([]float64, sampleVar.Card)

	// TODO: Find functions for sampleVar
	// TODO: Call each function with all vals for sampleVar and ADD all vals to sampleWeights

	// Convert sampleWeights from log space - and gather a total while we're at it
	totWeights := 0.0
	for i, w := range sampleWeights {
		v := math.Exp(w)
		totWeights += v
		sampleWeights[i] = v
	}

	// Select value based on the factor weights for our current variable
	r := g.rnd.Float64() * totWeights
	nextVal := -1
	for i, w := range sampleWeights {
		if r <= w {
			// Remember, it's an array of weights from 0 -> Card-1: we are
			// selecting an index based on those weights
			nextVal = i
			break
		}
		r -= w
	}

	if nextVal < 0 {
		return errors.Errorf("Failed to select a value from var %v, Exp(factor-weights)==%v", sampleVar.Name, sampleWeights)
	}

	// Update saved copy with new value and copy to caller's sample
	g.last[varIdx] = float64(nextVal)
	copy(s, g.last)

	return nil
}
