package main

import "github.com/CraigKelly/grample/cmd"

// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?
// TODO: need some benchmarks - and make sure to output

func main() {
	cmd.Execute()
}
