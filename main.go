package main

import "github.com/CraigKelly/grample/cmd"

// TODO: read evidence and use during sampling - see model/model.go

// TODO: need an initial adaptive-collapsing giibs sampler

// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?

// TODO: at least one unit test for cmd package - and maybe a benchmark?

// TODO: at least one unit test and one benchmark for rand

func main() {
	cmd.Execute()
}
