package sampler

import (
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

	for i := range ch.ChainHistory {
		ch.ChainHistory[i] = buffer.NewCircularInt(cw)
	}

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
		c.ChainHistory[varIdx].Add(value)

		c.TotalSampleCount++
	}

	return nil
}
