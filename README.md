---
title: grample README
subtitle: Sampling for Probabilistic Graphical Models
---

# Introduction

This software package is designed to read Markov networks and perform marginal
estimation using Gibbs sampling. The main motivation for creating Yet Another
MCMC software package was research: this is the experimental implementation of
Adaptive Rao-Blackwellisation as reported in
[Adaptive Rao-Blackwellisation in Gibbs Sampling for Probabilistic Graphical Models](http://proceedings.mlr.press/v89/kelly19a.html)

# Using and Citing

This code is licensed under the MIT license: see `LICENSE` for details. If you
use this code in a published work, please cite the paper
[here](http://proceedings.mlr.press/v89/kelly19a.html):

```tex
@inproceedings{AdaptiveRBGibbs,
  title={Adaptive Rao-Blackwellisation in Gibbs Sampling for Probabilistic Graphical Models},
  author={Kelly, Craig and Sarkhel, Somdeb and Venugopal, Deepak},
  booktitle={The 22nd International Conference on Artificial Intelligence and Statistics},
  editor = 	 {Chaudhuri, Kamalika and Sugiyama, Masashi},
  pages={2907--2915},
  year={2019},
  series = {Proceedings of Machine Learning Research},
}
```

# Installing and Running

There's no real installion. Use `go get -u github.com/CraigKelly/grample`
to get the latest code. From inside the grample directory, run `make`
to build. Then you can run `./grample -h` to get command line help.
You can see some examples in `./script/experiment`

# Using As a Library

If you want to grample as a library, that's fairly easy. There aren't
any directions right now, but see `./cmd/root.go` for examples. That's
our main command line implementation, so you can get a good idea of
how to use the sampler package.

# Dependencies

As of this writing, this code has only been tested with Go 1.10 and Go 1.11
(and minor versions).

Currently we are using `dep` for dependency management, so see `Gopkg.toml`
and `./vendor`. The short story is that we don't have many dependencies,
but we *are* using `github.com/spf13/cobra` to manage the command line
and `github.com/stretchr/testify` for unit test assertions.

# Hacking

Use the Makefile, which delegates to scripts located in `./scripts`.
