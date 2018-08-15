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
// sampled at least ConvergeWindow times.
func (c *Chain) AdvanceChain(wg *sync.WaitGroup) error {
	cwThresh := make([]int64, len(c.ChainHistory))

	for i, hist := range c.ChainHistory {
		cwThresh[i] = hist.TotalSeen + int64(c.ConvergenceWindow) + 1
	}

	keepRunning := func() bool {
		for i, hist := range c.ChainHistory {
			if c.Target.Vars[i].FixedVal < 0 && hist.TotalSeen < cwThresh[i] {
				return true
			}
		}
		return false
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		// While there is work to do, take {var count} samples
		for keepRunning() {
			for range c.Target.Vars {
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

		c.Target.Vars[varIdx].Marginal[value] += 1.0
		c.ChainHistory[varIdx].Add(value)

		c.TotalSampleCount++
	}

	return nil
}
