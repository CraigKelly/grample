package main

import "github.com/CraigKelly/grample/cmd"

// TODO: need an initial adaptive gibbs sampler (on var selection)
// TODO: need a collapsing Gibbs sample: both static(at begin) or dynamic(during chain)
// TODO: combine adaptive/collapse options

// TODO: trace file should include final scores for easy read back once we get started

// TODO: at least one unit test for cmd package - and maybe a benchmark?

// TODO: at least one unit test and one benchmark for rand

// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?

// TODO: web frontend for monitor so we can track what's going on?

func main() {
	cmd.Execute()
}
