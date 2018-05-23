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
var useEvidence bool
var solFile string
var samplerName string
var randomSeed int64
var burnIn int64
var maxIters int64
var maxSecs int64
var sampleRate float64
var traceFile string

// Helper for outputting parameters
func startupParms() {
	fmt.Printf("Verbose:        %v\n", verbose)
	fmt.Printf("Model:          %s\n", uaiFile)
	fmt.Printf("Apply Evidence: %v\n", useEvidence)
	fmt.Printf("Solution:       %s\n", solFile)
	fmt.Printf("Sampler:        %s\n", samplerName)
	fmt.Printf("Burn In:        %12d\n", burnIn)
	fmt.Printf("Max Iters:      %12d\n", maxIters)
	fmt.Printf("Max Secs:       %12d\n", maxSecs)
	fmt.Printf("Accept Rate:    %12.4f\n", sampleRate)
	fmt.Printf("Rnd Seed:       %12d\n", randomSeed)
}

// Help text for root command
const cmdHelp = `grample provides sampling-based inference for PGM's. Features include:

- The ability to read UAI PGM files (for models and evidence)
- A Gibbs sampler
- An experimental version of an Adaptive Gibbs sampler
`

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var cmd = &cobra.Command{
		Use:   "grample",
		RunE:  rootRunE,
		Short: "(Probalistic) Graphical Model Sampling Methods",
		Long:  cmdHelp,
	}

	pf := cmd.PersistentFlags()
	pf.BoolVarP(&verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")
	pf.Int64VarP(&randomSeed, "seed", "e", 1, "Random seed to use")
	pf.StringVarP(&uaiFile, "model", "m", "", "UAI model file to read")
	pf.BoolVarP(&useEvidence, "evidence", "d", false, "Apply evidence from evidence file (name inferred from model file")
	pf.StringVarP(&solFile, "solution", "o", "", "UAI MAR solution file to use for scoring")
	pf.StringVarP(&samplerName, "sampler", "s", "", "Name of sampler to use")
	pf.Int64VarP(&burnIn, "burnin", "b", -1, "Burn-In iteration count - if < 0, will use 2000*n (n= # vars)")
	pf.Int64VarP(&maxIters, "maxiters", "i", 0, "Maximum iterations (not including burnin) 0 if < 0 will use 20000*n")
	pf.Int64VarP(&maxSecs, "maxsecs", "x", 300, "Maximum seconds to run (0 for no maximum)")
	pf.StringVarP(&traceFile, "trace", "t", "", "Optional trace file: all samples written here")
	pf.Float64VarP(&sampleRate, "srate", "r", -1.0, "Rate at which samples are accepted (1.0 to accept all) - if < 0, will use 1/n")

	cmd.MarkPersistentFlagRequired("model")
	cmd.MarkPersistentFlagRequired("sampler")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Run/entry for the root cmd
func rootRunE(cmd *cobra.Command, args []string) error {
	fmt.Printf("grample\n")

	// Extra checks on parameters
	if sampleRate > 1.0 {
		return errors.Errorf("Invalid sample rate %v: must be in the range (0.0, 1.0)", sampleRate)
	}

	return modelMarginals()
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
	mod, err = model.NewModelFromFile(reader, uaiFile, useEvidence)
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

		totScore, maxScore, err := sol.AbsError(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calculation init score on startup")
		}
		hellScore, err := sol.HellingerError(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calc init hellinger on startup")
		}

		fmt.Printf("Start TotAE: %.6f nlog=%.3f\n", totScore, -math.Log(totScore))
		fmt.Printf("Start MaxAE: %.6f nlog=%.3f\n", maxScore, -math.Log(maxScore))
		fmt.Printf("Start HellE: %.6f nlog=%.3f\n", hellScore, -math.Log(hellScore))
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
	fmt.Printf("Main Sampling Start\n")

	// Note that our first status will happen faster than all later updates
	stopTime := startTime.Add(time.Duration(maxSecs) * time.Second)
	untilStatus := time.Duration(5) * time.Second
	nextStatus := startTime.Add(untilStatus / 2)

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
				// Only update marginal counts if this isn't a fixed var (evidence)
				if v.FixedVal < 0 {
					v.Marginal[oneSample[i]] += 1.0
				}
			}
		}

		// Time checking and status updates
		now := time.Now()
		if maxSecs > 0 && now.After(stopTime) {
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
				score, maxScore, err := sol.AbsError(mod)
				if err != nil {
					return errors.Wrapf(err, "Error calculating TAE")
				}
				evalReport = fmt.Sprintf("%8.6f nlog=%.3f (maxE %8.6f)", score, -math.Log(score), maxScore)
			}

			fmt.Printf(
				"  Iterations: %12d | Samples: %12d | Run time %12.2fsec | Eval %s\n",
				it, sampleCount, time.Now().Sub(startTime).Seconds(), evalReport,
			)
		}
	}

	// COMPLETED! normalize our marginals
	for _, v := range mod.Vars {
		if v.FixedVal >= 0 {
			v.NormMarginal()
		}
	}

	// TODO: write to a UAI MAR file

	// Output the marginals we found and our final evaluation
	fmt.Printf("DONE\n")

	// Output evidence vars first, then output vars we're estimating
	if verbose {
		for _, v := range mod.Vars {
			if v.FixedVal >= 0 {
				fmt.Printf("Variable[%d] %s (Card:%d, %+v) EVID=%d\n", v.ID, v.Name, v.Card, v.State, v.FixedVal)
			}
		}
		for _, v := range mod.Vars {
			if v.FixedVal < 0 {
				fmt.Printf("Variable[%d] %s (Card:%d, %+v) %+v\n", v.ID, v.Name, v.Card, v.State, v.Marginal)
			}
		}
	}

	if trace != nil {
		traceObj.SetIndent("", "  ")
		traceObj.Encode(mod)
	}

	if len(solFile) > 0 {
		totScore, maxScore, err := sol.AbsError(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calculating AE!")
		}
		hellScore, err := sol.HellingerError(mod)
		if err != nil {
			return errors.Wrapf(err, "Error calculating Hellinger Err!")
		}
		fmt.Printf("Final TotAE: %.6f nlog=%.3f\n", totScore, -math.Log(totScore))
		fmt.Printf("Final MaxAE: %.6f nlog=%.3f\n", maxScore, -math.Log(maxScore))
		fmt.Printf("Final HellE: %.6f nlog=%.3f\n", hellScore, -math.Log(hellScore))
	}

	return nil
}
