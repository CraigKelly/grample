package sampler

import (
	"math"
	"sync"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/pkg/errors"
)

// GibbsSimple is our baseline, simple to code Gibbs sampler
type GibbsSimple struct {
	gen         *rand.Generator
	pgm         *model.Model
	varSelector VarSampler
	varFuncs    map[int][]*model.Function
	last        []int
	valuePool   *sync.Pool
	varPool     *sync.Pool
}

// TODO: unit test error handling and getting at least one good sample

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(gen *rand.Generator, m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	// Select variable uniformly at random at each step
	uniform, err := NewUniformSampler(gen)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create uniform sampler in Gibbs Simple sample")
	}

	// We need an array of inys the size of our variables a LOT and run
	// parallel chains. This pool keeps our allocations low.
	varPool := &sync.Pool{
		New: func() interface{} {
			return make([]int, len(m.Vars))
		},
	}

	// We also use an array for holding counts across a variable's cardinality
	// (NOTE that these actually *ARE* floats)
	maxCard := 0
	for _, v := range m.Vars {
		if v.Card > maxCard {
			maxCard = v.Card
		}
	}
	valuePool := &sync.Pool{
		New: func() interface{} {
			return make([]float64, maxCard)
		},
	}

	s := &GibbsSimple{
		gen:         gen,
		pgm:         m,
		varSelector: uniform,
		varFuncs:    make(map[int][]*model.Function),
		last:        make([]int, len(m.Vars)),
		valuePool:   valuePool,
		varPool:     varPool,
	}

	// Set up functions: use log space for factors and keep track of functions
	// that involve each variable
	for _, f := range m.Funcs {
		err := f.UseLogSpace()
		if err != nil {
			return nil, errors.Wrapf(err, "Could not convert function %v to Log Space", f.Name)
		}

		for _, v := range f.Vars {
			s.varFuncs[v.ID] = append(s.varFuncs[v.ID], f)
		}
	}

	// Starting point in the sample space - note that the next call to Sample
	// will return the next sample, and not this one so our user will never
	// see this starting point unless they explicitly look for it.
	for i, v := range s.pgm.Vars {
		// Init any variable state that we track
		v.State["Selections"] = 0.0 // Number of times selected for sampling

		// Check on pgm vars to make sure they are set up the way we expect
		if i != v.ID {
			// ID should match index in PGM model
			return nil, errors.Errorf("Invalid ID for var %s: expected %d but was %d", v.Name, v.ID, i)
		}
		if len(s.varFuncs[v.ID]) < 1 {
			// If variable not in single factor, then can't be sampled
			return nil, errors.Errorf("There are no functions for var %s (ID=%d)", v.Name, v.ID)
		}

		// Select value for every variable with uniform prob
		val, err := uniform.ValSample(v.Card)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not generate start sample for variable %v", v.Name)
		}

		s.last[i] = val
	}

	return s, nil
}

// Sample returns a single sample - implements FullSampler
func (g *GibbsSimple) Sample(s []int) error {
	if len(s) != len(g.pgm.Vars) {
		return errors.Errorf("Sample size %d != Var size %d in model %s", len(s), len(g.pgm.Vars), g.pgm.Name)
	}

	// Select next variable to sample
	varIdx, err := g.varSelector.VarSample(g.pgm.Vars)
	if err != nil {
		return errors.Wrapf(err, "Could not sample from vars in model %s", g.pgm.Name)
	}
	sampleVar := g.pgm.Vars[varIdx]
	sampleVar.State["Selections"] += 1.0

	// Find all related factors and marginalize for sampleVar

	// We are going to gather up the result of the functions across all the
	// values for our variable (sampleVar)
	sampleWeightsBuffer := g.valuePool.Get().([]float64)
	defer g.valuePool.Put(sampleWeightsBuffer)
	sampleWeights := sampleWeightsBuffer[:sampleVar.Card]
	// Don't forget to zero the weights since we're reusing buffers
	for i := range sampleWeights {
		sampleWeights[i] = 0.0
	}

	// For each function/factor that our variable is involved with...
	callValBuffer := g.varPool.Get().([]int)
	defer g.varPool.Put(callValBuffer)
	for _, fun := range g.varFuncs[sampleVar.ID] {
		// Set up call values: we want a slice of the correct size. We
		// initialize with values from our last sample. We also need to find
		// the index for sampleVar in this list.
		callVals := callValBuffer[:len(fun.Vars)]
		callIdx := -1
		for i, v := range fun.Vars {
			callVals[i] = g.last[v.ID]
			if v.ID == sampleVar.ID {
				callIdx = i // Found our variable!
			}
		}

		if callIdx < 0 {
			return errors.Errorf("Var %d:%s not in function %s var list?!",
				sampleVar.ID, sampleVar.Name, fun.Name,
			)
		}

		// Now we need to call once for every value possible for our current
		// variable
		for v := 0; v < sampleVar.Card; v++ {
			callVals[callIdx] = v
			result, err := fun.Eval(callVals)
			if err != nil {
				return errors.Wrapf(err, "Error generating a sample on function %s with selected variable %d:%s",
					fun.Name, sampleVar.ID, sampleVar.Name,
				)
			}
			sampleWeights[v] += result
		}
	}

	// Convert sampleWeights from log space - and gather a total while we're at it
	totWeights := 0.0
	for i, w := range sampleWeights {
		v := math.Exp(w)
		totWeights += v
		sampleWeights[i] = v
	}

	// Select value based on the factor weights for our current variable
	r := g.gen.Float64() * totWeights
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
	g.last[varIdx] = nextVal
	copy(s, g.last)

	return nil
}
