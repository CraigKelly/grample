package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/CraigKelly/grample/sampler"
)

// We want to cheat as little as possible, so we grab the start time ASAP
var startTime = time.Now()

// Parameters
var verbose bool
var uaiFile string
var solFile string
var samplerName string
var randomSeed int64
var burnIn int64
var maxIters int64
var maxSecs int64
var sampleRate float64
var traceFile string

func startupParms() {
	fmt.Printf("Verbose:     %v\n", verbose)
	fmt.Printf("Model:       %s\n", uaiFile)
	fmt.Printf("Solution:    %s\n", solFile)
	fmt.Printf("Sampler:     %s\n", samplerName)
	fmt.Printf("Burn In:     %12d\n", burnIn)
	fmt.Printf("Max Iters:   %12d\n", maxIters)
	fmt.Printf("Max Secs:    %12d\n", maxSecs)
	fmt.Printf("Accept Rate: %12.4f\n", sampleRate)
	fmt.Printf("Rnd Seed:    %12d\n", randomSeed)

}

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

			if sampleRate > 1.0 {
				return errors.Errorf("Invalid sample rate %v: must be in the range (0.0, 1.0)", sampleRate)
			}

			return modelMarginals()
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")

	rootCmd.PersistentFlags().Int64VarP(&randomSeed, "seed", "e", 1, "Random seed to use")

	rootCmd.PersistentFlags().StringVarP(&uaiFile, "model", "m", "", "UAI model file to read")
	rootCmd.PersistentFlags().StringVarP(&solFile, "solution", "o", "", "UAI MAR solution file to use for scoring")
	rootCmd.PersistentFlags().StringVarP(&samplerName, "sampler", "s", "", "Name of sampler to use")
	rootCmd.PersistentFlags().Int64VarP(&burnIn, "burnin", "b", -1, "Burn-In iteration count - if < 0, will use 2000*n (n= # vars)")
	rootCmd.PersistentFlags().Int64VarP(&maxIters, "maxiters", "i", 0, "Maximum iterations (not including burnin) 0 if < 0 will use 20000*n")
	rootCmd.PersistentFlags().Int64VarP(&maxSecs, "maxsecs", "x", 300, "Maximum seconds to run (0 for no maximum)")
	rootCmd.PersistentFlags().StringVarP(&traceFile, "trace", "t", "", "Optional trace file: all samples written here")
	rootCmd.PersistentFlags().Float64VarP(&sampleRate, "srate", "r", -1.0, "Rate at which samples are accepted (1.0 to accept all) - if < 0, will use 1/n")

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
	var sol *model.Solution
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

	// Read solution file (if we have one)
	if len(solFile) > 0 {
		sol, err = model.NewSolutionFromFile(reader, solFile)
		if err != nil {
			return errors.Wrapf(err, "Could not read solution file %s", solFile)
		}

		score, err := sol.Score(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calculation init score on startup")
		}

		fmt.Printf("Starting eval metric (worst case): %.6f nlog=%.3f\n", score, -math.Log(score))
	}

	// Some of our parameters are based on variable count
	if sampleRate <= 0.0 {
		sampleRate = 1.0 / float64(len(mod.Vars))
	}
	if burnIn < 0 {
		burnIn = int64(2000 * len(mod.Vars))
	}
	if maxIters < 0 {
		maxIters = int64(20000 * len(mod.Vars))
	}

	// Report what's going on
	startupParms()

	// Create our concurrent PRNG
	gen, err := rand.NewGenerator(randomSeed)
	if err != nil {
		return errors.Wrapf(err, "Could not create Generator from seed %d", randomSeed)
	}

	// select sampler
	if strings.ToLower(samplerName) == "gibbssimple" {
		samp, err = sampler.NewGibbsSimple(gen, mod)
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
	var traceObj *json.Encoder
	if len(traceFile) > 0 {
		trace, err = os.Create(traceFile)
		if err != nil {
			return errors.Wrapf(err, "Could not open trace file %s", traceFile)
		}
		traceObj = json.NewEncoder(trace)
		if verbose {
			fmt.Printf("WARNING: verbose is set, every accepted sample will be written to trace file %s\n", traceFile)
		}
	}

	// Sampling: main iterations
	// TODO: parallel chains
	fmt.Printf("Main Sampling Start\n")

	stopTime := startTime.Add(time.Duration(maxSecs) * time.Second)
	untilStatus := time.Duration(2) * time.Second
	nextStatus := startTime.Add(untilStatus)

	it := int64(1)
	sampleCount := int64(0)
	keepWorking := true
	for keepWorking {
		err = samp.Sample(oneSample)
		if err != nil {
			return errors.Wrapf(err, "Error during main iteration it %d", it)
		}

		// Only trace and update marginals if we accept the sample
		if gen.Float64() <= sampleRate {
			sampleCount++

			if trace != nil && verbose {
				traceObj.Encode(oneSample)
			}

			for i, v := range mod.Vars {
				v.Marginal[oneSample[i]] += 1.0
			}
		}

		// Time checking and status updates
		now := time.Now()
		if maxSecs > 0 && now.After(stopTime) {
			fmt.Printf("Reached stop time %v\n", stopTime)
			keepWorking = false
		}

		// Don't forget to check iterations!
		it++
		if maxIters > 0 && it > maxIters {
			keepWorking = false
		}

		// Status update
		if now.After(nextStatus) || !keepWorking {
			nextStatus = now.Add(untilStatus)

			evalReport := "---"
			if len(solFile) > 1 {
				score, err := sol.Score(mod)
				if err != nil {
					return errors.Wrapf(err, "Error calculating score")
				}
				evalReport = fmt.Sprintf("%8.6f nlog=%.3f", score, -math.Log(score))
			}

			fmt.Printf(
				"  Iterations: %12d | Samples: %12d | Run time %12.2fsec | Eval %s\n",
				it, sampleCount, time.Now().Sub(startTime).Seconds(), evalReport,
			)
		}
	}

	// Output the marginals we found and our final evaluation
	// TODO: write to a UAI MAR file
	fmt.Printf("DONE\n")

	for _, v := range mod.Vars {
		v.NormMarginal()

		if verbose {
			fmt.Printf("Variable[%d] %s (Card:%d, %+v) %+v\n", v.ID, v.Name, v.Card, v.State, v.Marginal)
		}
	}

	if trace != nil {
		traceObj.SetIndent("", "  ")
		traceObj.Encode(mod)
	}

	if len(solFile) > 0 {
		score, err := sol.Score(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calculating final score! Will continue: error %+v", err)
		}
		fmt.Printf("Final eval metric (worst case): %.6f nlog=%.3f\n", score, -math.Log(score))
	}

	return nil
}
