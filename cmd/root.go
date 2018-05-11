package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/sampler"
)

var verbose bool
var uaiFile string
var samplerName string
var randomSeed int64
var burnIn int64
var maxIters int64
var traceFile string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "grample",
		Short: "(Probalistic) Graphical Model Sampling Methods",
		Long: `grample provides sampling-based inference for PGM's. Features include:

  - The ability to read UAI PGM files (for models and evidence)
  - A Gibbs sampler
  - An experimental version of an Adaptive Gibbs sampler
    `,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("grample\n")
			fmt.Printf("Verbose:   %v\n", verbose)
			fmt.Printf("Model:     %s\n", uaiFile)
			fmt.Printf("Sampler:   %s\n", samplerName)
			fmt.Printf("Burn In:   %12d\n", burnIn)
			fmt.Printf("Max Iters: %12d\n", maxIters)
			fmt.Printf("Rnd Seed:  %12d\n", randomSeed)

			rand.Seed(randomSeed)

			return modelMarginals()
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")

	rootCmd.PersistentFlags().StringVarP(&uaiFile, "model", "m", "", "UAI model file to read")
	rootCmd.PersistentFlags().StringVarP(&samplerName, "sampler", "s", "", "Name of sampler to use")
	rootCmd.PersistentFlags().Int64VarP(&burnIn, "burnin", "b", 500, "Burn-In iteration count")
	rootCmd.PersistentFlags().Int64VarP(&maxIters, "maxiters", "i", 20000, "Maximum iterations (not including burnin)")
	rootCmd.PersistentFlags().StringVarP(&traceFile, "trace", "t", "", "Optional trace file: all samples written here")

	rootCmd.MarkPersistentFlagRequired("model")
	rootCmd.MarkPersistentFlagRequired("sampler")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Our current default action (and the only one we support)
func modelMarginals() error {
	var mod *model.Model
	var err error
	var samp sampler.FullSampler

	// Read model from file
	fmt.Printf("Reading model from %s\n", uaiFile)
	reader := model.UAIReader{}
	mod, err = model.NewModelFromFile(reader, uaiFile)
	if err != nil {
		return err
	}
	if verbose {
		fmt.Printf("MODEL: %+v\n", mod)
		for _, v := range mod.Vars {
			fmt.Printf("  %+v\n", v)
		}
		for _, f := range mod.Funcs {
			fmt.Printf("  %+v\n", f)
		}
	}

	// select sampler
	if strings.ToLower(samplerName) == "gibbssimple" {
		samp, err = sampler.NewGibbsSimple(rand.NewSource(rand.Int63()), mod)
		if err != nil {
			return errors.Wrapf(err, "Could not create %s", samplerName)
		}
	} else {
		return errors.Errorf("Unknown Sampler: %s", samplerName)
	}

	// Sampling: burn in
	oneSample := make([]int, len(mod.Vars))

	fmt.Printf("Performing burn-in (%d)\n", burnIn)
	for it := int64(1); it <= burnIn; it++ {
		err = samp.Sample(oneSample)
		if err != nil {
			return errors.Wrapf(err, "Error during burn in on it %d", it)
		}
	}

	// Trace file
	var trace *os.File
	if len(traceFile) > 0 {
		trace, err = os.Create(traceFile)
		if err != nil {
			return errors.Wrapf(err, "Could not open trace file %s", traceFile)
		}
	}

	// Sampling: main iterations
	// TODO: also have max time elapsed
	// TODO: parallel chains
	// TODO: single chain - only sample every x samples?
	fmt.Printf("Sampling until max iter %d\n", maxIters)
	for it := int64(1); it <= maxIters; it++ {
		err = samp.Sample(oneSample)
		if err != nil {
			return errors.Wrapf(err, "Error during main iteration it %d", it)
		}

		// Write trace if necessary
		if trace != nil {
			fmt.Fprintf(trace, "%v\n", oneSample)
		}

		// Update variable marginals
		for i, v := range mod.Vars {
			v.Marginal[oneSample[i]] += 1.0
		}

		// TODO: make output time based OR iteration based
		// TODO: iteration count and time elapsed gets smaller if verbose
		if it%5000 == 0 {
			fmt.Printf("  Iterations: %12d\n", it)
		}
	}

	// Output the marginals we found
	// TODO: write to a UAI MAR file
	// TODO: output comparison to a previous MAR file (should be known good)
	fmt.Printf("Done. Marginals:\n")
	for _, v := range mod.Vars {
		fmt.Printf("Variable[%d] %s (Card:%d, SelCount:%d)\n", v.ID, v.Name, v.Card, v.Counter)
		v.NormMarginal()
		fmt.Printf("%+v\n", v.Marginal)
	}

	return nil
}
