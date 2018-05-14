package main

import "github.com/CraigKelly/grample/cmd"

// TODO: mersenne twister rand num generator with consistent seeding - not
//       protected for concur. Current plan is to switch to a buffered
//       channel and a goroutine that just pumps rand numbers at it.
//       That way we can get single-seed determinism across multiple cores
//       (easily).
// TODO: checkpointing for chains (so we can freeze and continue) - which means
//       model/sampler/chain all need to be included?
// TODO: need some benchmarks - and make sure to output

func main() {
	cmd.Execute()
}
