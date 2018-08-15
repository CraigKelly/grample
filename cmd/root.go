package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/CraigKelly/grample/sampler"
)

// We want to cheat as little as possible, so we grab the start time ASAP
var startTime = time.Now()

// Parameter
type startupParams struct {
	verbose        bool
	uaiFile        string
	useEvidence    bool
	solFile        bool
	samplerName    string
	randomSeed     int64
	burnIn         int64
	convergeWindow int64
	maxIters       int64
	maxSecs        int64
	traceFile      string

	// These are created/handled by Setup
	out    *log.Logger
	verb   *log.Logger
	trace  *log.Logger
	traceJ JSONLogger
	mon    *monitor
}

// JSONLogger is a simple interface for JSON logging (matches json.Encoder) and
// nil/no-op implementation
type JSONLogger interface {
	Encode(v interface{}) error
	SetIndent(prefix, indent string)
}

// DiscardJSON does nothing
type DiscardJSON struct{}

// Encode for DiscardJSON does nothing
func (n *DiscardJSON) Encode(interface{}) error {
	return nil
}

// SetIndent for DiscardJSON does nothing
func (n *DiscardJSON) SetIndent(string, string) {
	return
}

// Setup handles initialization based on supplied parameters
func (s *startupParams) Setup() error {
	s.out = log.New(os.Stdout, "", 0)

	if s.verbose {
		s.verb = log.New(os.Stdout, "", 0)
	} else {
		s.verb = log.New(ioutil.Discard, "", 0)
	}

	if len(s.traceFile) > 0 {
		f, err := os.Create(s.traceFile)
		if err != nil {
			return err
		}
		s.trace = log.New(f, "", 0)
		s.traceJ = json.NewEncoder(f)
	} else {
		s.trace = log.New(ioutil.Discard, "", 0)
		s.traceJ = &DiscardJSON{}
	}

	s.mon = &monitor{}

	return nil
}

// Report just writes commands - must be called after Setup
func (s *startupParams) Report() {
	s.out.Printf("Verbose:        %v\n", s.verbose)
	s.out.Printf("Model:          %s\n", s.uaiFile)
	s.out.Printf("Apply Evidence: %v\n", s.useEvidence)
	s.out.Printf("Solution:       %v\n", s.solFile)
	s.out.Printf("Sampler:        %s\n", s.samplerName)
	s.out.Printf("Burn In:        %12d\n", s.burnIn)
	s.out.Printf("Converge Win:   %12d\n", s.convergeWindow)
	s.out.Printf("Max Iters:      %12d\n", s.maxIters)
	s.out.Printf("Max Secs:       %12d\n", s.maxSecs)
	s.out.Printf("Rnd Seed:       %12d\n", s.randomSeed)
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
	sp := &startupParams{}

	rootRunE := func(cmd *cobra.Command, args []string) error {
		err := sp.Setup()
		if err != nil {
			return err
		}
		sp.out.Printf("grample\n")

		err = sp.mon.Start()
		if err != nil {
			return err
		}
		defer sp.mon.Stop()

		return modelMarginals(sp)
	}

	var cmd = &cobra.Command{
		Use:   "grample",
		RunE:  rootRunE,
		Short: "(Probalistic) Graphical Model Sampling Methods",
		Long:  cmdHelp,
	}

	pf := cmd.PersistentFlags()
	pf.BoolVarP(&sp.verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")
	pf.Int64VarP(&sp.randomSeed, "seed", "e", 1, "Random seed to use")
	pf.StringVarP(&sp.uaiFile, "model", "m", "", "UAI model file to read")
	pf.BoolVarP(&sp.useEvidence, "evidence", "d", false, "Apply evidence from evidence file (name inferred from model file")
	pf.BoolVarP(&sp.solFile, "solution", "o", false, "Use UAI MAR solution file to score (name inferred from model file)")
	pf.StringVarP(&sp.samplerName, "sampler", "s", "", "Name of sampler to use")
	pf.Int64VarP(&sp.burnIn, "burnin", "b", -1, "Burn-In iteration count - if < 0, will use 2000*n (n= # vars)")
	pf.Int64VarP(&sp.convergeWindow, "cwin", "w", -1, "Sample window size for measuring convergence, if <= 0 will use burnin size")
	pf.Int64VarP(&sp.maxIters, "maxiters", "i", 0, "Maximum iterations (not including burnin) 0 if < 0 will use 20000*n")
	pf.Int64VarP(&sp.maxSecs, "maxsecs", "x", 300, "Maximum seconds to run (0 for no maximum)")
	pf.StringVarP(&sp.traceFile, "trace", "t", "", "Optional trace file: all samples written here")

	cmd.MarkPersistentFlagRequired("model")
	cmd.MarkPersistentFlagRequired("sampler")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Our current default action (and the only one we support)
func modelMarginals(sp *startupParams) error {
	var mod *model.Model
	var sol *model.Solution
	var err error

	// Read model from file
	sp.out.Printf("Reading model from %s\n", sp.uaiFile)
	reader := model.UAIReader{}
	mod, err = model.NewModelFromFile(reader, sp.uaiFile, sp.useEvidence)
	if err != nil {
		return err
	}

	// Helper func for writing out an error suite of scoring
	var errorBuffer strings.Builder
	errorReport := func(prefix string, es *model.ErrorSuite, short bool) {
		sp.mon.LastMeanHellinger.Set(es.MeanHellinger)
		sp.mon.LastMaxHellinger.Set(es.MaxHellinger)
		sp.mon.LastMeanJSD.Set(es.MeanJSDiverge)
		sp.mon.LastMaxJSD.Set(es.MaxJSDiverge)

		if short {
			errorBuffer.Reset()
			patt := "%s=>%.6f(%7.3f),X%.6f(%7.3f) | "
			fmt.Fprintf(
				&errorBuffer, patt, "MAE",
				es.MeanMeanAbsError, -math.Log2(es.MeanMeanAbsError),
				es.MaxMeanAbsError, -math.Log2(es.MaxMeanAbsError),
			)
			fmt.Fprintf(
				&errorBuffer, patt, "XAE",
				es.MeanMaxAbsError, -math.Log2(es.MeanMaxAbsError),
				es.MaxMaxAbsError, -math.Log2(es.MaxMaxAbsError),
			)
			fmt.Fprintf(
				&errorBuffer, patt, "HEL",
				es.MeanHellinger, -math.Log2(es.MeanHellinger),
				es.MaxHellinger, -math.Log2(es.MaxHellinger),
			)
			fmt.Fprintf(
				&errorBuffer, patt, "JSD",
				es.MeanJSDiverge, -math.Log2(es.MeanJSDiverge),
				es.MaxJSDiverge, -math.Log2(es.MaxJSDiverge),
			)
			sp.out.Printf(errorBuffer.String())
			return
		}

		sp.out.Printf("%s ... M:mean(neg log), X:max(neg log)\n", prefix)
		patt := "%15s => M:%.6f(%7.3f) X:%.6f(%7.3f)\n"
		sp.out.Printf(
			patt, "MeanAbsError",
			es.MeanMeanAbsError, -math.Log2(es.MeanMeanAbsError),
			es.MaxMeanAbsError, -math.Log2(es.MaxMeanAbsError),
		)
		sp.out.Printf(
			patt, "MaxAbsError",
			es.MeanMaxAbsError, -math.Log2(es.MeanMaxAbsError),
			es.MaxMaxAbsError, -math.Log2(es.MaxMaxAbsError),
		)
		sp.out.Printf(
			patt, "Hellinger",
			es.MeanHellinger, -math.Log2(es.MeanHellinger),
			es.MaxHellinger, -math.Log2(es.MaxHellinger),
		)
		sp.out.Printf(
			patt, "JS Diverge",
			es.MeanJSDiverge, -math.Log2(es.MeanJSDiverge),
			es.MaxJSDiverge, -math.Log2(es.MaxJSDiverge),
		)
	}

	// Read solution file (if we have one)
	if sp.solFile {
		solFilename := sp.uaiFile + ".MAR"
		sol, err = model.NewSolutionFromFile(reader, solFilename)
		if err != nil {
			return errors.Wrapf(err, "Could not read solution file %s", solFilename)
		}

		score, err := sol.Error(mod.Vars)
		if err != nil {
			return errors.Wrapf(err, "Error calculating init score on startup")
		}
		errorReport("START", score, false)
	}

	// Some of our parameters are based on variable count
	if sp.burnIn < 0 {
		sp.burnIn = int64(2000 * len(mod.Vars))
	}
	if sp.convergeWindow < 0 {
		sp.convergeWindow = sp.burnIn
	}
	if sp.maxIters < 0 {
		sp.maxIters = int64(20000 * len(mod.Vars))
	}

	// Report what's going on
	sp.Report()
	sp.mon.BurnIn.Set(sp.burnIn)
	sp.mon.ConvergeWindow.Set(sp.convergeWindow)
	sp.mon.MaxIters.Set(sp.maxIters)
	sp.mon.MaxSeconds.Set(sp.maxSecs)

	// Create our concurrent PRNG
	gen, err := rand.NewGenerator(sp.randomSeed)
	if err != nil {
		return errors.Wrapf(err, "Could not create Generator from seed %d", sp.randomSeed)
	}

	// Create chains and do burnin
	sp.out.Printf("Creating chains and performing burn-in (%d)\n", sp.burnIn)

	chains := make([]*sampler.Chain, 2)
	for idx := range chains {
		modCopy := mod.Clone()

		var samp sampler.FullSampler
		if strings.ToLower(sp.samplerName) == "gibbssimple" {
			samp, err = sampler.NewGibbsSimple(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
		} else {
			return errors.Errorf("Unknown Sampler: %s", sp.samplerName)
		}

		ch, err := sampler.NewChain(modCopy, samp, int(sp.convergeWindow), sp.burnIn)
		if err != nil {
			return errors.Wrapf(err, "Could not create initial chain")
		}
		chains[idx] = ch
	}

	// Trace file warning - it can get huge in verbose mode
	if len(sp.traceFile) > 0 && sp.verbose {
		sp.out.Printf("WARNING: verbose is set, every accepted sample will be written to trace file %s\n", sp.traceFile)
	}

	// Sampling: main iterations
	sp.out.Printf("Main Sampling Start\n")

	// Note that our first status will happen faster than all later updates
	stopTime := startTime.Add(time.Duration(sp.maxSecs) * time.Second)
	untilStatus := time.Duration(5) * time.Second
	nextStatus := startTime.Add(untilStatus / 2)

	wg := sync.WaitGroup{}

	// MAIN LOOP
	keepWorking := true
	for keepWorking {
		for _, ch := range chains {
			ch.AdvanceChain(&wg)
		}
		wg.Wait()

		// Time checking and status updates
		now := time.Now()
		if sp.maxSecs > 0 && now.After(stopTime) {
			keepWorking = false
		}

		// Don't forget to check iterations for quit
		sampleCount := chains[0].TotalSampleCount
		sp.mon.Iterations.Set(sampleCount)
		if sp.maxIters > 0 && sampleCount > sp.maxIters {
			keepWorking = false
		}

		// Status update
		if now.After(nextStatus) || !keepWorking {
			nextStatus = now.Add(untilStatus)

			runTime := time.Now().Sub(startTime).Seconds()
			sp.mon.RunTime.Set(runTime)
			sp.out.Printf("  Samps: %12d | RT %12.2fsec\n", sampleCount, runTime)

			if sp.solFile {
				merged, err := sampler.MergeChains(chains)
				if err != nil {
					return errors.Wrapf(err, "Could not merge chains to calculate score")
				}
				score, err := sol.Error(merged)
				if err != nil {
					return errors.Wrapf(err, "Error calculating score")
				}
				errorReport("", score, true)
			}
		}
	}

	// COMPLETED! normalize our marginals
	finalVars, err := sampler.MergeChains(chains)
	if err != nil {
		return errors.Wrapf(err, "Error in final chain merge")
	}
	for _, v := range finalVars {
		v.NormMarginal()
	}

	// Output the marginals we found and our final evaluation
	sp.out.Printf("DONE\n")

	// Write score if we have a solution file
	if sp.solFile {
		score, err := sol.Error(finalVars)
		if err != nil {
			return errors.Wrapf(err, "Error calculating Final Score!")
		}
		errorReport("FINAL", score, false)

		// Update the state map for variables for the trace/verbose stuff below
		for i, v := range finalVars {
			s := sol.Vars[i]
			for c := 0; c < v.Card; c++ {
				ky := fmt.Sprintf("MAR[%d]", c)
				v.State[ky] = s.Marginal[c]
			}
		}
	}

	// Trace file and verbose output for final results
	// Output evidence vars first, then output vars we're estimating
	sp.traceJ.SetIndent("", "")
	sp.trace.Printf("// EVIDENCE")
	for _, v := range finalVars {
		if v.FixedVal >= 0 {
			sp.traceJ.Encode(v)
			sp.verb.Printf("Variable[%d] %s (Card:%d, %+v) EVID=%d\n", v.ID, v.Name, v.Card, v.State, v.FixedVal)
		}
	}
	sp.trace.Printf("// VARS (ESTIMATED)")
	for _, v := range finalVars {
		if v.FixedVal < 0 {
			sp.traceJ.Encode(v)
			sp.verb.Printf("Variable[%d] %s (Card:%d, %+v) %+v\n", v.ID, v.Name, v.Card, v.State, v.Marginal)
		}
	}

	sp.trace.Printf("// ENTIRE MODEL\n")
	sp.traceJ.SetIndent("", "  ")
	sp.traceJ.Encode(mod)

	return nil
}
