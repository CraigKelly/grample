package main

import "github.com/CraigKelly/grample/cmd"

// TODO: need a convergence metric - will start with G-R

// TODO: need an initial adaptive gibbs sampler (on var selection)
// TODO: need a collapsing Gibbs sample: both static(at begin) or dynamic(during chain)
// TODO: combine adaptive/collapse options

// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?

// TODO: at least one unit test for cmd package - and maybe a benchmark?

// TODO: at least one unit test and one benchmark for rand

// TODO: trace file should include final scores for easy read back once we get started

func main() {
	cmd.Execute()
}
