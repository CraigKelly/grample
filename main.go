package main

import "github.com/CraigKelly/grample/cmd"

// TODO: Grid 12x12 is basically the best that can be calc'ed dynamically, so get grids 12 and 18

// TODO: need an initial adaptive gibbs sampler (on var selection)
// TODO: need a collapsing Gibbs sample: both static(at begin) or dynamic(during chain)
// TODO: combine adaptive/collapse options

// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?

// TODO: at least one unit test for cmd package - and maybe a benchmark?

// TODO: at least one unit test and one benchmark for rand

func main() {
	cmd.Execute()
}
