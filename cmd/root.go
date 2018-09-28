package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/CraigKelly/grample/sampler"
)

// TODO: actually test new code, ESP adaptive sampler

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
	baseCount      int64
	maxIters       int64
	maxSecs        int64
	traceFile      string
	monitorAddr    string

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

func (s *startupParams) dump(out *log.Logger) {
	out.Printf("Verbose:        %v\n", s.verbose)
	out.Printf("Model:          %s\n", s.uaiFile)
	out.Printf("Apply Evidence: %v\n", s.useEvidence)
	out.Printf("Solution:       %v\n", s.solFile)
	out.Printf("Sampler:        %s\n", s.samplerName)
	out.Printf("Burn In:        %12d\n", s.burnIn)
	out.Printf("Converge Win:   %12d\n", s.convergeWindow)
	out.Printf("Num Base Chain: %12d\n", s.baseCount)
	out.Printf("Max Iters:      %12d\n", s.maxIters)
	out.Printf("Max Secs:       %12d\n", s.maxSecs)
	out.Printf("Rnd Seed:       %12d\n", s.randomSeed)
	out.Printf("Monitor Addr:   %s\n", s.monitorAddr)
}

// Report just writes commands - must be called after Setup
func (s *startupParams) Report() {
	s.dump(s.out)
}

// Trace writes a report to the trace output
func (s *startupParams) Trace() {
	s.dump(s.trace)
}

// Help text for root command
const cmdHelp = `grample provides sampling-based inference for PGM's. Features include:

- The ability to read UAI PGM files (for models and evidence)
- A Gibbs sampler
- An experimental version of an Adaptive Gibbs sampler
`

type grampleCmd func(*startupParams) error

func runGrampleCmd(sp *startupParams, f grampleCmd) error {
	err := sp.Setup()
	if err != nil {
		return err
	}

	sp.out.Printf("grample\n")

	err = sp.mon.Start(sp.monitorAddr)
	if err != nil {
		return err
	}

	defer sp.mon.Stop()

	return f(sp)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	sp := &startupParams{}

	var cmd = &cobra.Command{
		Use:   "grample",
		Short: "(Probalistic) Graphical Model Sampling Methods",
		Long:  cmdHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGrampleCmd(sp, modelMarginals)
		},
	}

	var collapseCmd = &cobra.Command{
		Use:   "collapse",
		Short: "Single-Variable Collapse Checking for a Model",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGrampleCmd(sp, CollapsedIteration)
		},
	}

	cmd.AddCommand(collapseCmd)

	pf := cmd.PersistentFlags()
	pf.BoolVarP(&sp.verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")
	pf.Int64VarP(&sp.randomSeed, "seed", "e", 0, "Random seed to use")
	pf.StringVarP(&sp.uaiFile, "model", "m", "", "UAI model file to read")
	pf.BoolVarP(&sp.useEvidence, "evidence", "d", false, "Apply evidence from evidence file (name inferred from model file")
	pf.BoolVarP(&sp.solFile, "solution", "o", false, "Use UAI MAR solution file to score (name inferred from model file)")
	pf.StringVarP(&sp.samplerName, "sampler", "s", "", "Name of sampler to use (simple, collapsed, adaptive)")
	pf.Int64VarP(&sp.burnIn, "burnin", "b", -1, "Burn-In iteration count - if < 0, will use 2000*n (n= # vars)")
	pf.Int64VarP(&sp.convergeWindow, "cwin", "w", -1, "Sample window size for measuring convergence, if <= 0 will use burnin size")
	pf.Int64VarP(&sp.baseCount, "chains", "c", -1, "Number of base/starting chains, if <= 0 will use number of CPUs")
	pf.Int64VarP(&sp.maxIters, "maxiters", "i", 0, "Maximum iterations (not including burnin) 0 if < 0 will use 20000*n")
	pf.Int64VarP(&sp.maxSecs, "maxsecs", "x", 300, "Maximum seconds to run (0 for no maximum)")
	pf.StringVarP(&sp.traceFile, "trace", "t", "", "Optional trace file: all samples written here")
	pf.StringVarP(&sp.monitorAddr, "addr", "a", ":8000", "Address (ip:port) that the monitor will listen at")

	cmd.MarkPersistentFlagRequired("model")
	cmd.MarkPersistentFlagRequired("sampler")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// // Handy reporting of current error/distance. Yes, global errorBuffer that is used by
// this function that we assume is never called concurrently.
var errorBuffer strings.Builder

func errorReport(sp *startupParams, prefix string, es *model.ErrorSuite, short bool) {
	// Update monitor with latest error results
	sp.mon.LastMeanHellinger.Set(es.MeanHellinger)
	sp.mon.LastMaxHellinger.Set(es.MaxHellinger)
	sp.mon.LastMeanJSD.Set(es.MeanJSDiverge)
	sp.mon.LastMaxJSD.Set(es.MaxJSDiverge)

	// Select
	var patt string
	var titles []string

	if short {
		patt = "%s=>%.6f(%7.3f),X%.6f(%7.3f) | "
		titles = []string{"MAE", "XAE", "HEL", "JSD"}
	} else {
		sp.out.Printf("%s ... M:mean(neg log), X:max(neg log)\n", prefix)
		patt = "%15s => M:%.6f(%7.3f) X:%.6f(%7.3f)\n"
		titles = []string{"MeanAbsError", "MaxAbsError", "Hellinger", "JS Diverge"}
	}

	// Use an error buffer so that we get to choose when a \n is logged
	errorBuffer.Reset()

	fmt.Fprintf(
		&errorBuffer, patt, titles[0],
		es.MeanMeanAbsError, -math.Log2(es.MeanMeanAbsError),
		es.MaxMeanAbsError, -math.Log2(es.MaxMeanAbsError),
	)
	fmt.Fprintf(
		&errorBuffer, patt, titles[1],
		es.MeanMaxAbsError, -math.Log2(es.MeanMaxAbsError),
		es.MaxMaxAbsError, -math.Log2(es.MaxMaxAbsError),
	)
	fmt.Fprintf(
		&errorBuffer, patt, titles[2],
		es.MeanHellinger, -math.Log2(es.MeanHellinger),
		es.MaxHellinger, -math.Log2(es.MaxHellinger),
	)
	fmt.Fprintf(
		&errorBuffer, patt, titles[3],
		es.MeanJSDiverge, -math.Log2(es.MeanJSDiverge),
		es.MaxJSDiverge, -math.Log2(es.MaxJSDiverge),
	)

	sp.out.Printf(errorBuffer.String())
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
	sp.out.Printf("Model has %d vars and %d functions\n", len(mod.Vars), len(mod.Funcs))

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
		errorReport(sp, "START", score, false)
	}

	// Some of our parameters are based on variable count
	if sp.randomSeed < 1 {
		n := time.Now()
		sp.randomSeed = int64(n.Second()) + int64(n.Nanosecond()) + int64(n.Minute())
	}
	if sp.burnIn < 0 {
		sp.burnIn = int64(2000 * len(mod.Vars))
	}
	if sp.convergeWindow <= 0 {
		sp.convergeWindow = sp.burnIn
	}
	if sp.maxIters < 0 {
		sp.maxIters = int64(20000 * len(mod.Vars))
	}
	if sp.baseCount <= 0 {
		sp.baseCount = int64(runtime.NumCPU())
		if sp.baseCount < 2 {
			sp.out.Printf("Base chain count was %d, forcing to 2\n", sp.baseCount)
			sp.baseCount = 2
		}
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

	chains := make([]*sampler.Chain, sp.baseCount)

	for idx := range chains {
		sp.out.Printf(" ... Chain %3d out of %3d\n", idx+1, sp.baseCount)
		modCopy := mod.Clone()

		var samp sampler.FullSampler

		if strings.ToLower(sp.samplerName) == "simple" {
			samp, err = sampler.NewGibbsSimple(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
		} else if strings.ToLower(sp.samplerName) == "collapsed" {
			coll, err := sampler.NewGibbsCollapsed(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
			colVar, err := coll.Collapse(-1)
			if err != nil {
				return errors.Wrapf(err, "Could not collapse random var on startup")
			}
			sp.out.Printf("        - Collaped variable %v:%v\n", colVar.ID, colVar.Name)
			sp.out.Printf("MARGINAL: %+v\n", colVar.Marginal)
			samp = coll
		} else if strings.ToLower(sp.samplerName) == "adaptive" {
			coll, err := sampler.NewGibbsCollapsed(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
			samp = coll
		} else {
			return errors.Errorf("Unknown Sampler: %s", sp.samplerName)
		}

		ch, err := sampler.NewChain(modCopy, samp, int(sp.convergeWindow), sp.burnIn)
		if err != nil {
			return errors.Wrapf(err, "Could not create initial chain")
		}

		chains[idx] = ch
		sp.mon.BaseChains.Add(1)
		sp.mon.TotalChains.Add(1)
	}

	// Chains created: now we can select our adaptive strategy
	var adapt sampler.AdaptiveSampler
	if strings.ToLower(sp.samplerName) == "adaptive" {
		// Adapt based on convergence metric
		adapt, err = sampler.NewConvergenceSampler(mod.Clone())
	} else {
		// Everything just skips adaptation
		adapt, err = sampler.NewIdentitySampler()
	}
	if err != nil {
		return errors.Wrapf(err, "Could not create adaptation strategy for %s", sp.samplerName)
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

	keepAdapting := true
	noAdaptTime := startTime.Add(time.Duration(sp.maxSecs/2) * time.Second)

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
		sampleCount := int64(0)
		for _, ch := range chains {
			sampleCount += ch.TotalSampleCount
		}
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
				errorReport(sp, "", score, true)
			}
		}

		// Adaptive update (if we're still updating)
		if keepAdapting {
			if now.After(noAdaptTime) {
				sp.out.Printf("STOPPING ADAPTATION\n")
				keepAdapting = false
			}
		}
		if keepWorking && keepAdapting {
			chains, err = adapt.Adapt(chains)
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
		errorReport(sp, "FINAL", score, false)

		// Update the state map for variables for the trace/verbose stuff below
		for i, v := range finalVars {
			s := sol.Vars[i]
			for c := 0; c < v.Card; c++ {
				ky := fmt.Sprintf("SOL-MAR[%d]", c)
				v.State[ky] = s.Marginal[c]
			}
		}

		// Go ahead and include Merlin info if we can find a merlin file
		merlinFilename := sp.uaiFile + ".merlin.MAR"
		if _, err := os.Stat(merlinFilename); !os.IsNotExist(err) {
			merlin, re := model.NewSolutionFromFile(reader, merlinFilename)
			if re != nil {
				return errors.Wrapf(re, "Found merlin MAR file but could not read it")
			}

			merlinError, re := merlin.Error(sol.Vars)
			if re != nil {
				return errors.Wrapf(re, "Error calculating merlin error")
			}
			errorReport(sp, "MERLIN SCORE", merlinError, false)

			merlinError, re = merlin.Error(finalVars)
			if re != nil {
				return errors.Wrapf(re, "Error calculating merlin error")
			}
			errorReport(sp, "OUR SCORE USING MERLIN AS SOLUTION", merlinError, false)
		}
	}

	// Get final convergence scores (and individual errors)
	hellConverge, err := sampler.ChainConvergence(chains, model.HellingerDiff)
	if err != nil {
		return errors.Wrapf(err, "Error getting final Hellinger Convergence")
	}
	jsConverge, err := sampler.ChainConvergence(chains, model.JSDivergence)
	if err != nil {
		return errors.Wrapf(err, "Error getting final JS Convergence")
	}
	maxaeConverge, err := sampler.ChainConvergence(chains, model.MaxAbsDiff)
	if err != nil {
		return errors.Wrapf(err, "Error getting final MaxAbsDiff Convergence")
	}
	avgaeConverge, err := sampler.ChainConvergence(chains, model.MeanAbsDiff)
	if err != nil {
		return errors.Wrapf(err, "Error getting final MeanAbsDiff Convergence")
	}

	for i, v := range finalVars {
		v.State["Hell-Convergence"] = hellConverge[i]
		v.State["JS-Convergence"] = jsConverge[i]
		v.State["MaxAD-Convergence"] = maxaeConverge[i]
		v.State["AvgAD-Convergence"] = avgaeConverge[i]
		if sp.solFile {
			v.State["Hell-Error"] = model.HellingerDiff(v, sol.Vars[i])
			v.State["JS-Error"] = model.JSDivergence(v, sol.Vars[i])
			v.State["MaxAD-Error"] = model.MaxAbsDiff(v, sol.Vars[i])
			v.State["AvgAD-Error"] = model.MeanAbsDiff(v, sol.Vars[i])
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

	sp.trace.Printf("// OPERATING PARAMS\n")
	sp.Trace()

	sp.trace.Printf("// ENTIRE MODEL\n")
	sp.traceJ.SetIndent("", "  ")
	sp.traceJ.Encode(mod)

	return nil
}
