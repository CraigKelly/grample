package sampler

import (
	"fmt"
	"math"

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

	s := &GibbsCollapsed{
		baseSampler:  base,
		varNeighbors: nil,
	}

	err = s.FunctionsChanged()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// FunctionsChanged is called when the models Function array has changed. That
// means we need to update some of our bookkeeping.
func (g *GibbsCollapsed) FunctionsChanged() error {
	base := g.baseSampler

	// A lookup from variables to their neighbors
	neighbors := make([]varSet, len(base.pgm.Vars))

	// Create a neighbor entry per variable
	for i, v := range base.pgm.Vars {
		if i != v.ID {
			return errors.Errorf("Invalid variable setup: [%d] => %+v", i, v)
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

	g.varNeighbors = neighbors
	return nil
}

// NeighborVarMax is the max size of the neighborhood allowed for a
// variable that we will collapse. Note that it includes the variable itself,
// so the total size of input space is 2^(M-1) where M is NeighborVarMax.
const NeighborVarMax = 8

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

	// Get our target variable - note that we clone the variable and zero the
	// marginal for summing up below
	collVar := pgm.Vars[varIdx].Clone()
	if collVar.FixedVal >= 0 {
		return nil, errors.Errorf("Can not collapsed Fixed Val variable %v:%v", collVar.ID, collVar.Name)
	}
	if collVar.Collapsed {
		return nil, errors.Errorf("Already collapsed variable %v:%v", collVar.ID, collVar.Name)
	}
	for i := 0; i < collVar.Card; i++ {
		collVar.Marginal[i] = 1e-6 // We start small instead of just a zero value
	}

	// IMPORTANT: remember in our blanket array, the variable index is NO
	// LONGER EQUAL to v.ID.  That's why we need an xref: we can get to an
	// index in blanket (and varState defined below) from a variable ID via
	// blanketXref.
	blanket := make([]*model.Variable, 0, len(pgm.Vars))
	blanketXref := make(map[int]int)
	for vi, inBlanket := range g.varNeighbors[varIdx] {
		if inBlanket {
			v := pgm.Vars[vi]
			blanket = append(blanket, pgm.Vars[vi])
			blanketXref[v.ID] = len(blanket) - 1
		}
	}

	// Get all the functions we'll need to collapse and precreate a cross-ref.
	// We'll also check our functions to make sure everything is OK
	funcs := g.baseSampler.varFuncs[varIdx]
	funcNameRef := make(map[string]bool)
	for _, f := range funcs {
		funcNameRef[f.Name] = true
		if !f.IsLog {
			return nil, errors.Errorf("Function %v is not set up for Log Space", f.Name)
		}
	}

	// We will be creating a new function from our collapsing work
	postFunc, err := model.NewFunction(len(pgm.Funcs), blanket)
	if err != nil {
		return nil, err
	}
	// We also override the name
	postFunc.Name = fmt.Sprintf("COLLAPSE-%v", collVar.Name)

	// We need a buffer to call each function AND a buffer to iterate function values
	callValBuffer := base.varPool.Get().([]int)
	defer base.varPool.Put(callValBuffer)

	varState := base.varPool.Get().([]int)
	defer base.varPool.Put(varState)

	// We also need a buffer for function results
	funcResultsBuffer := base.valuePool.Get().([]float64)
	defer base.valuePool.Put(funcResultsBuffer)
	funcResults := funcResultsBuffer[:collVar.Card]

	// Iterate over all configurations in the blanket/neighborhood
	varIter, err := model.NewVariableIter(blanket)
	if err != nil {
		return nil, err
	}
	for {
		err := varIter.Val(varState)
		if err != nil {
			return nil, err
		}

		// Iterate over all functions, updating varState
		for i := range funcResults {
			funcResults[i] = 0.0
		}

		// We need a total result for our new function as well
		totResult := 0.0

		for _, fun := range base.varFuncs[collVar.ID] {
			// Populate call value slice and save where our collapsed variable is
			callVals := callValBuffer[:len(fun.Vars)]
			marginalVal := -1
			for i, v := range fun.Vars {
				stateIdx := blanketXref[v.ID]
				callVals[i] = varState[stateIdx]
				if v.ID == collVar.ID {
					marginalVal = callVals[i]
				}
			}
			if marginalVal < 0 {
				return nil, errors.Errorf("Func %v - could not find the variable being marginalized", fun.Name)
			}

			// Call function and add (in log space, so really mutliply) to our
			// function results.
			result, err := fun.Eval(callVals)
			if err != nil {
				return nil, errors.Wrapf(err, "Collapsing error calling function %v (%+v)", fun.Name, callVals)
			}

			funcResults[marginalVal] += result
			totResult += result
		}

		// Now update our marginal with the final function result. Remember
		// that we need to convert from log space first
		for i, val := range funcResults {
			if math.Abs(val) > 1e-7 {
				collVar.Marginal[i] += math.Exp(val)
			}
		}

		// Also update our new function
		if math.Abs(totResult) > 1e-7 {
			postFunc.AddValue(varState, math.Exp(totResult))
		}

		// Time for next variable state
		if !varIter.Next() {
			break
		}
	}

	// We have now collected an entire marginal
	err = collVar.NormMarginal()
	if err != nil {
		return nil, err
	}

	// We also have a new function
	err = postFunc.UseLogSpace()
	if err != nil {
		return nil, err
	}

	// Add our new function and delete the replaced functions
	pgm.Funcs = append(pgm.Funcs, postFunc)

	insert := -1
	for i, f := range pgm.Funcs {
		if ok, del := funcNameRef[f.Name]; ok && del {
			continue // We want to delete this function
		}
		insert++
		if insert != i {
			pgm.Funcs[insert] = pgm.Funcs[i]
		}
	}
	if insert < 0 {
		return nil, errors.Errorf("No functions left after collapse!")
	}
	pgm.Funcs = pgm.Funcs[:insert+1]

	// Now we need to update internal tracking: both in this sampler and in the
	// base/simple sampler. We also need to re-run model checking to make sure
	// we haven't broken anything
	err = base.FunctionsChanged()
	if err != nil {
		return nil, err
	}
	err = g.FunctionsChanged()
	if err != nil {
		return nil, err
	}
	err = pgm.Check()
	if err != nil {
		return nil, err
	}

	// All done - update the variable itself from our cloned copy and return
	// our results
	dest := pgm.Vars[varIdx]
	dest.Collapsed = true
	copy(dest.Marginal, collVar.Marginal)
	return dest, nil
}

// Sample returns a single sample - implements FullSampler
func (g *GibbsCollapsed) Sample(s []int) (int, error) {
	base := g.baseSampler
	pgm := base.pgm

	if len(s) != len(pgm.Vars) {
		return -1, errors.Errorf("Samples size %d is wrong", len(s))
	}

	// Note excludeCollapsed=True
	varIdx, err := base.varSelector.VarSample(pgm.Vars, false)
	if err != nil {
		return -1, err
	}

	// Our function updates above mean that both collapsed and un-collapsed
	// variables can now be sampled by the simple sampler
	return base.SampleVar(varIdx, s)
}
