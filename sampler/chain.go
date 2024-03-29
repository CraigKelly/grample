package sampler

import (
	"math"
	"sync"

	"github.com/CraigKelly/grample/buffer"
	"github.com/CraigKelly/grample/model"
	"github.com/pkg/errors"
)

// Chain provides functionality around a Gibbs sampler.
type Chain struct {
	Target            *model.Model
	Sampler           FullSampler
	ConvergenceWindow int
	ChainHistory      []*buffer.CircularInt
	TotalSampleCount  int64
	LastSample        []int
}

// Measure is an error metric used by ChainConverge. One example is our
// model.HellingerDiff
type Measure func(v1 *model.Variable, v2 *model.Variable) float64

// ChainConvergence returns an array of floats that corresponds to the array of
// variables in the chains. Each float is a measure of the current convergence
// of the specified variable, where values close to 1.0 are better. Currently
// we return 1.0 for any collapsed variable. Note that as an optimization,
// this function will accept variables that are pre-merged. If an empty array
// is passed, variables will be merged automatically
func ChainConvergence(chains []*Chain, distFunc Measure, mergedVars []*model.Variable) ([]float64, error) {
	if len(chains) < 2 {
		return nil, errors.Errorf("Convergence requires at least 2 chains")
	}

	var err error

	// We need a merged chain so that we have an overall distribution
	if len(mergedVars) < 1 {
		mergedVars, err = MergeChains(chains)
		if err != nil {
			return nil, err
		}
	}

	// our actual values that we'll need
	vals := make([]float64, len(mergedVars))

	// values we can calculate before starting
	sampleCount := float64(chains[0].ConvergenceWindow)
	chainCount := float64(len(chains))

	// for B (between-chain) calcs
	bNorm := sampleCount / (chainCount - 1)

	// for vhat calculations
	wFactor := (sampleCount - 1) / sampleCount
	bFactor := (chainCount + 1) / (chainCount * sampleCount)

	for i, v := range mergedVars {
		// Variables that are fixed from evidence OR that are collapsed have already converged
		if v.Collapsed || v.FixedVal >= 0 {
			vals[i] = 1.0
			continue
		}

		// Find the within-chain and between-chain distance/error
		W := 1e-8 // within-chain
		B := 1e-8 // between-chain
		for _, ch := range chains {
			ch.ChainHistory[i].FirstHalf()
			ch.ChainHistory[i].SecondHalf()

			wOne, bOne, err := ch.ChainDist(distFunc, i, v)
			if err != nil {
				return nil, err
			}

			W += wOne
			B += bOne
		}
		W /= chainCount
		B *= bNorm

		// Final: calcuate V-hat and the adjusted PSRF
		vhat := (wFactor * W) + (bFactor * B)
		vals[i] = math.Sqrt((4.0 * vhat) / (2.0 * W))
	}

	return vals, nil
}

// MergeChains returns a single variable array from multiple chains suitable
// for marginal dist calculations.
func MergeChains(chains []*Chain) ([]*model.Variable, error) {
	chLen := len(chains)
	if chLen < 1 {
		return nil, errors.Errorf("Can not merge 0 chains")
	}
	if chLen == 1 {
		return chains[0].Target.Vars, nil
	}

	// If variable is collapsed in any chain, use that single var's Marginal as
	// the merged estimate. If there is NOT a collapsed variable, then we start
	// with the variable in the first chain
	collapsedVars := make(map[int]bool)
	varLen := len(chains[0].Target.Vars)
	vars := make([]*model.Variable, varLen)

	var found *model.Variable
	for varIdx := 0; varIdx < varLen; varIdx++ {
		found = nil
		for _, ch := range chains {
			v := ch.Target.Vars[varIdx]
			if v.Collapsed {
				found = v
				break
			}
		}

		if found != nil {
			collapsedVars[varIdx] = true
			vars[varIdx] = found.Clone()
		} else {
			collapsedVars[varIdx] = false
			vars[varIdx] = chains[0].Target.Vars[varIdx].Clone()
		}
	}

	for _, ch := range chains[1:] {
		if len(ch.Target.Vars) != varLen {
			return nil, errors.Errorf("Cannot merge chain with %d vars into %d vars", len(ch.Target.Vars), varLen)
		}
		for varIdx, src := range ch.Target.Vars {
			if isCollapsed, inMap := collapsedVars[varIdx]; inMap && isCollapsed {
				continue // No summation for already collapsed vars
			}
			for marIdx, val := range src.Marginal {
				vars[varIdx].Marginal[marIdx] += val
			}
		}
	}

	// All done - ready to send back marged results
	return vars, nil
}

// NewChain returns a chain ready to go. It even performs burnin.
func NewChain(mod *model.Model, samp FullSampler, cw int, burnIn int64) (*Chain, error) {
	ch := &Chain{
		Target:            mod,
		Sampler:           samp,
		ConvergenceWindow: cw,
		ChainHistory:      make([]*buffer.CircularInt, len(mod.Vars)),
		TotalSampleCount:  0,
		LastSample:        make([]int, len(mod.Vars)),
	}

	// Create all the buffers we need
	for i := range ch.ChainHistory {
		ch.ChainHistory[i] = buffer.NewCircularInt(cw)
	}

	// Perform requested burn-in
	for i := int64(0); i < burnIn; i++ {
		err := ch.oneSample(false)
		if err != nil {
			return nil, errors.Wrap(err, "Failure during chain burn in")
		}
	}

	return ch, nil
}

// AdvanceChain asynchonously generates samples until all variables have been
// sampled at least ConvergeWindow times. Variables that are Fixed or Collapsed
// are not checked for ConvergeWindow times
func (c *Chain) AdvanceChain(wg *sync.WaitGroup) error {
	cwThresh := make([]int64, len(c.ChainHistory))

	for i, hist := range c.ChainHistory {
		cwThresh[i] = hist.TotalSeen + int64(c.ConvergenceWindow) + 1
	}

	keepRunning := func() bool {
		for i, hist := range c.ChainHistory {
			v := c.Target.Vars[i]
			if !v.Collapsed && v.FixedVal < 0 && hist.TotalSeen < cwThresh[i] {
				return true
			}
		}
		return false
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		// If we have N variables, we should take at least N samples before
		// checking to see if we need to keep working. However, as a simple
		// optmization we currently run for 2N.
		batchSize := len(c.Target.Vars) * 2

		// While there is work to do, take {var count} samples
		for keepRunning() {
			for i := 0; i < batchSize; i++ {
				err := c.oneSample(true)
				if err != nil {
					panic("Async sample generation failed - cannot continue")
				}
			}
		}
	}()

	return nil
}

// oneSample takes a single sample and optionally updates the chain state.
func (c *Chain) oneSample(updateVars bool) error {
	varIdx, err := c.Sampler.Sample(c.LastSample)
	if err != nil {
		return errors.Wrap(err, "Error taking sample")
	}
	if varIdx < 0 || c.Target.Vars[varIdx].FixedVal >= 0 {
		return errors.New("Invalid sample")
	}

	if updateVars {
		value := c.LastSample[varIdx]

		v := c.Target.Vars[varIdx]
		if !v.Collapsed {
			c.Target.Vars[varIdx].Marginal[value] += 1.0
		}
		err := c.ChainHistory[varIdx].Add(value)
		if err != nil {
			return errors.Wrap(err, "Error taking sample and adding to ChainHistory")
		}

		c.TotalSampleCount++
	}

	return nil
}

// ChainDist returns the within-chain and between-chain error based for the
// given Measure function against the specified variable.
// varIdx is the index of the variable under consideration
// mergedVar is a variable represented the merged chain estimate of the marginal
// The returned tuple is (within-chain, between-chain)
func (c *Chain) ChainDist(distFunc Measure, varIdx int, mergedVar *model.Variable) (float64, float64, error) {
	hist := c.ChainHistory[varIdx]
	if hist.TotalSeen < int64(c.ConvergenceWindow) {
		return math.NaN(), math.NaN(), errors.Errorf("Total seen %d < Convergence Window %d", hist.TotalSeen, c.ConvergenceWindow)
	}

	vsrc := c.Target.Vars[varIdx]
	if vsrc.Card != mergedVar.Card {
		return math.NaN(), math.NaN(), errors.Errorf("Variable mismatch")
	}

	v1 := vsrc.Clone()
	v2 := vsrc.Clone()

	for i := range vsrc.Marginal {
		v1.Marginal[i] = 1e-8
		v2.Marginal[i] = 1e-8
	}

	for iter := hist.FirstHalf(); iter.Next(); {
		val := iter.Value()
		v1.Marginal[val] += 1.0
	}
	for iter := hist.SecondHalf(); iter.Next(); {
		val := iter.Value()
		v2.Marginal[val] += 1.0
	}

	within := distFunc(v1, v2)

	// Collapse v2 into v1 for chain marginal estimate
	for i, val := range v2.Marginal {
		v1.Marginal[i] += val
	}
	between := distFunc(mergedVar, v1)

	return within, between, nil
}
