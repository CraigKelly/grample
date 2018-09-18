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
	weighted    WeightedSampler
	varFuncs    map[int][]*model.Function
	last        []int
	valuePool   *sync.Pool
	varPool     *sync.Pool
}

// NewGibbsSimple creates a new sampler
func NewGibbsSimple(gen *rand.Generator, m *model.Model) (*GibbsSimple, error) {
	if m == nil {
		return nil, errors.New("No model supplied")
	}

	// Select variable uniformly at random at each step
	uniform, err := NewUniformSampler(gen, len(m.Vars))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create uniform sampler in Gibbs Simple sample")
	}

	// We need an array of ints the size of our variables a LOT and run
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
		weighted:    uniform,
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

		// Select value for every variable with uniform prob UNLESS it is Fixed
		// (in that case we always know the value)
		if v.FixedVal >= 0 {
			s.last[i] = v.FixedVal
		} else {
			val, err := uniform.UniSample(v.Card)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not generate start sample for variable %v", v.Name)
			}
			s.last[i] = val
		}
	}

	return s, nil
}

// FunctionsChanged is called when the models Function array has changed. That
// means we need to update some of our bookkeeping.
func (g *GibbsSimple) FunctionsChanged() error {
	g.varFuncs = make(map[int][]*model.Function)

	for _, f := range g.pgm.Funcs {
		if !f.IsLog {
			return errors.Errorf("Function %v is not in log space on FunctionsChanged", f.Name)
		}
		for _, v := range f.Vars {
			g.varFuncs[v.ID] = append(g.varFuncs[v.ID], f)
		}
	}

	// Now we need to reset our last position (we do this instead of burnin)
	for i, v := range g.pgm.Vars {
		if v.FixedVal >= 0 {
			g.last[i] = v.FixedVal
		} else {
			val, err := g.weighted.WeightedSample(v.Card, v.Marginal)
			if err != nil {
				return errors.Wrapf(err, "Could no generated a start sample for var %v", v.Name)
			}
			g.last[i] = val
		}
	}

	return nil
}

// Sample returns a single sample - implements FullSampler
func (g *GibbsSimple) Sample(s []int) (int, error) {
	if len(s) != len(g.pgm.Vars) {
		return -1, errors.Errorf("Sample size %d != Var size %d in model %s", len(s), len(g.pgm.Vars), g.pgm.Name)
	}

	// Select next variable to sample
	varIdx, err := g.varSelector.VarSample(g.pgm.Vars, false)
	if err != nil {
		return -1, errors.Wrapf(err, "Could not sample from vars in model %s", g.pgm.Name)
	}

	return g.SampleVar(varIdx, s)
}

// SampleVar samples from the pre-selected varIdx variable.
func (g *GibbsSimple) SampleVar(varIdx int, s []int) (int, error) {
	sampleVar := g.pgm.Vars[varIdx]
	sampleVar.State["Selections"] += 1.0

	if sampleVar.FixedVal >= 0 {
		return -1, errors.Errorf("Selected sample variable %v which has FixedVal=%d", sampleVar.Name, sampleVar.FixedVal)
	}

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
			return -1, errors.Errorf(
				"Var %d:%s not in function %s var list?!",
				sampleVar.ID, sampleVar.Name, fun.Name,
			)
		}

		// Now we need to call once for every value possible for our current
		// variable. Remember that we're in log space, so we can add values
		// (instead of multiplying the function results)
		for v := 0; v < sampleVar.Card; v++ {
			callVals[callIdx] = v
			result, err := fun.Eval(callVals)
			if err != nil {
				return -1, errors.Wrapf(err,
					"Error generating a sample on function %s with selected variable %d:%s",
					fun.Name, sampleVar.ID, sampleVar.Name,
				)
			}
			sampleWeights[v] += result
		}
	}

	// Convert sampleWeights from log space - and gather a total while we're at it
	// To make sure everything remains stable, we scale our numbers up. Recalling
	// that adding in log-space is equivalent to multiplication, we just add a constant
	// to all weights if the minimum weight is too low. There is mainly for numerical
	// stability, but it also helps with debugging things like our min weight check below
	minWeight := sampleWeights[0]
	for _, w := range sampleWeights[1:] {
		if w < minWeight {
			minWeight = w
		}
	}
	if minWeight < -8.0 {
		for i, w := range sampleWeights {
			sampleWeights[i] = w - (minWeight - 1.5) // should all be positive now
		}
	}

	totWeights := 0.0
	for i, w := range sampleWeights {
		v := math.Exp(w)
		totWeights += v
		sampleWeights[i] = v
	}

	// Remember that for Gibbs sampling to work, every option must be possible. As
	// a result, we make sure that no weight results in a prob < minProb
	for i, w := range sampleWeights {
		if w/totWeights < 1e-6 {
			delta := totWeights * 1e-6
			if delta <= 1e-12 {
				return -1, errors.Errorf("Logic error: w=%.12f, totw=%.12f, delta=%.12f",
					sampleWeights[i], totWeights, delta)
			}
			totWeights += delta // Adj total as well!
			sampleWeights[i] += delta
		}
	}

	// Select value based on the factor weights for our current variable and
	// then update saved copy with new value and copy to caller's sample.
	nextVal, err := g.weighted.WeightedSample(len(sampleWeights), sampleWeights)
	if err != nil {
		return -1, nil
	}

	g.last[varIdx] = nextVal
	copy(s, g.last)

	return varIdx, nil
}
