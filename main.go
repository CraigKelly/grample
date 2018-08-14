package main

import "github.com/CraigKelly/grample/cmd"

// TODO: need an initial adaptive gibbs sampler (on var selection)
// TODO: need a collapsing Gibbs sample: both static(at begin) or dynamic(during chain)
// TODO: combine adaptive/collapse options

// TODO: at least one unit test for cmd package - and maybe a benchmark?

func main() {
	cmd.Execute()
}
