package main

import "github.com/CraigKelly/grample/cmd"

// TODO: simple no evidence Gibbs sampling
// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?
// TODO: need some benchmarks - and make sure to output
// TODO: accept a starting seed, which is used to create a master rand source

func main() {
	cmd.Execute()
}
