package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
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
	baseCount      int64
	chainAdds      int64
	maxIters       int64
	maxSecs        int64
	traceFile      string
	monitorAddr    string
	experiment     bool

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

	return nil
}

func (s *startupParams) dump(out *log.Logger) {
	out.Printf("Verbose:                %v\n", s.verbose)
	out.Printf("Model:                  %s\n", s.uaiFile)
	out.Printf("Apply Evidence:         %v\n", s.useEvidence)
	out.Printf("Solution:               %v\n", s.solFile)
	out.Printf("Sampler:                %s\n", s.samplerName)
	out.Printf("Burn In:                %12d\n", s.burnIn)
	out.Printf("Converge Win:           %12d\n", s.convergeWindow)
	out.Printf("Num Base Chain:         %12d\n", s.baseCount)
	out.Printf("Chains Added per Adapt: %12d\n", s.chainAdds)
	out.Printf("Max Iters:              %12d\n", s.maxIters)
	out.Printf("Max Secs:               %12d\n", s.maxSecs)
	out.Printf("Rnd Seed:               %12d\n", s.randomSeed)
	out.Printf("Monitor Addr:           %s\n", s.monitorAddr)
	out.Printf("Experiment Mode:        %v\n", s.experiment)
}

// Report just writes commands - must be called after Setup
func (s *startupParams) Report() {
	s.dump(s.out)
}

// Trace writes a report to the trace output
func (s *startupParams) Trace() {
	s.dump(s.trace)
}

// During startup in command line mode, we will panic on various errors
func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
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

	if sp.mon != nil {
		err = sp.mon.Start(sp.monitorAddr)
		if err != nil {
			return err
		}

		defer sp.mon.Stop()
	}

	return f(sp)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	sp := &startupParams{}

	// ROOT command (no default action) and global flags
	var cmd = &cobra.Command{
		Use:   "grample",
		Short: "(Probalistic) Graphical Model Sampling Methods",
		Long:  cmdHelp,
	}

	pf := cmd.PersistentFlags()
	pf.BoolVarP(&sp.verbose, "verbose", "v", false, "Verbose logging (ALL samples written to --trace file)")
	pf.Int64VarP(&sp.randomSeed, "seed", "e", 0, "Random seed to use")
	pf.StringVarP(&sp.traceFile, "trace", "t", "", "Optional trace file")

	// IMPORTANT: note that startup params get changed based on the command.
	// For instance, sampler creates a monitor and collapse always turns on
	// solFile and useEvidence.

	// SAMPLER
	var sampleCmd = &cobra.Command{
		Use:   "sample",
		Short: "Gibbs sampling run",
		RunE: func(cmd *cobra.Command, args []string) error {
			sp.mon = &monitor{}
			return runGrampleCmd(sp, modelMarginals)
		},
	}

	cmd.AddCommand(sampleCmd)

	pf = sampleCmd.PersistentFlags()
	pf.StringVarP(&sp.samplerName, "sampler", "s", "", "Name of sampler to use (simple, collapsed, adaptive)")
	pf.StringVarP(&sp.uaiFile, "model", "m", "", "UAI model file to read")
	pf.BoolVarP(&sp.useEvidence, "evidence", "d", false, "Apply evidence from evidence file (name inferred from model file")
	pf.BoolVarP(&sp.solFile, "solution", "o", false, "Use UAI MAR solution file to score (name inferred from model file)")
	pf.Int64VarP(&sp.burnIn, "burnin", "b", -1, "Burn-In iteration count - if < 0, will use 2000*n (n= # vars)")
	pf.Int64VarP(&sp.convergeWindow, "cwin", "w", -1, "Sample window size for measuring convergence, if <= 0 will use burnin size")
	pf.Int64VarP(&sp.baseCount, "chains", "c", -1, "Number of base/starting chains, if <= 0 will use number of CPUs")
	pf.Int64VarP(&sp.chainAdds, "chainadds", "a", 1, "Number of chains added in an adaptive step (only valid if sampler=adaptive)")
	pf.Int64VarP(&sp.maxIters, "maxiters", "i", 0, "Maximum iterations (not including burnin) 0 if < 0 will use 20000*n")
	pf.Int64VarP(&sp.maxSecs, "maxsecs", "x", 300, "Maximum seconds to run (0 for no maximum)")
	pf.StringVarP(&sp.monitorAddr, "addr", "", ":8000", "Address (ip:port) that the monitor will listen at")
	pf.BoolVarP(&sp.experiment, "experiment", "p", false, "Experiment mode - every chain advance status is written to trace file")

	PanicIf(sampleCmd.MarkPersistentFlagRequired("model"))
	PanicIf(sampleCmd.MarkPersistentFlagRequired("sampler"))

	// COLLAPSE (collapse all available variables)
	var collapseCmd = &cobra.Command{
		Use:   "collapse",
		Short: "Single-Variable Collapse Checking for a Model with Evidence and Solution available",
		RunE: func(cmd *cobra.Command, args []string) error {
			sp.solFile = true
			sp.useEvidence = true
			return runGrampleCmd(sp, CollapsedIteration)
		},
	}

	cmd.AddCommand(collapseCmd)

	pf = collapseCmd.PersistentFlags()
	pf.StringVarP(&sp.uaiFile, "model", "m", "", "UAI model file (evidence and MAR files expected)")

	PanicIf(collapseCmd.MarkPersistentFlagRequired("model"))

	// DOT command
	var dotCmd = &cobra.Command{
		Use:   "dot",
		Short: "Output graphviz representation of the model",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGrampleCmd(sp, DotOutput)
		},
	}

	cmd.AddCommand(dotCmd)

	pf = dotCmd.PersistentFlags()
	pf.StringVarP(&sp.uaiFile, "model", "m", "", "UAI model file")

	PanicIf(dotCmd.MarkPersistentFlagRequired("model"))

	// Finally time time to execute
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// // Handy reporting of current error/distance. Yes, global errorBuffer that is used by
// this function that we assume is never called concurrently.
var errorBuffer strings.Builder

func errorReport(sp *startupParams, prefix string, es *model.ErrorSuite, short bool, target *log.Logger) {
	if target == nil {
		// Default log
		target = sp.out
		// Update monitor with latest error results
		// (Custom log target means we don't update the mmonitor)
		sp.mon.LastMeanHellinger.Set(es.MeanHellinger)
		sp.mon.LastMaxHellinger.Set(es.MaxHellinger)
		sp.mon.LastMeanJSD.Set(es.MeanJSDiverge)
		sp.mon.LastMaxJSD.Set(es.MaxJSDiverge)
	}

	// Select
	var patt string
	var titles []string

	if short {
		patt = "%s=>%.6f(%7.3f),X%.6f(%7.3f) | "
		titles = []string{"MAE", "XAE", "HEL", "JSD"}
	} else {
		target.Printf("%s ... M:mean(neg log), X:max(neg log)\n", prefix)
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

	target.Printf(errorBuffer.String())
}

// Our current default action (and the only one we support)
func modelMarginals(sp *startupParams) error {
	var mod *model.Model
	var sol *model.Solution
	var err error

	// Can't be in experiment mode with a trace file
	if sp.experiment && len(sp.traceFile) < 1 {
		return errors.New("Experiment mode requires a trace file")
	}

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
		errorReport(sp, "START", score, false, nil)
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
	}
	if sp.baseCount < 2 {
		sp.out.Printf("Base chain count was %d, forcing to 2\n", sp.baseCount)
		sp.baseCount = 2
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
			// Simple Gibbs - just created the chains we need
			samp, err = sampler.NewGibbsSimple(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
		} else if strings.ToLower(sp.samplerName) == "collapsed" {
			// Collapsed Gibbs - collapse a random variable per chain
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
			// Adaptive (collapsed) Gibbs - don't pre-collapse anything: the
			// adaptive sampler strategy will handle that for us
			coll, err := sampler.NewGibbsCollapsed(gen, modCopy)
			if err != nil {
				return errors.Wrapf(err, "Could not create %s", sp.samplerName)
			}
			samp = coll
		} else {
			// Doh! We don't know this sampler
			return errors.Errorf("Unknown Sampler: %s", sp.samplerName)
		}

		// Create our chains and update the monitor
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
		// Adapt based on convergence metric: we currently just use the the
		// samplers default Measure for convergence.
		adapt, err = sampler.NewConvergenceSampler(gen, mod.Clone(), nil)
	} else {
		// Everything just skips adaptation
		if sp.chainAdds != 1 {
			return errors.Errorf("Sampler is not adaptive: ChainAdds=%d makes no sense", sp.chainAdds)
		}
		adapt, err = sampler.NewIdentitySampler()
	}
	if err != nil {
		return errors.Wrapf(err, "Could not create adaptation strategy for %s", sp.samplerName)
	}

	// Trace file warning - it can get huge in verbose mode
	if len(sp.traceFile) > 0 && sp.verbose {
		sp.out.Printf("WARNING: verbose is set, every accepted sample will be written to trace file %s\n", sp.traceFile)
	}

	// If in experiment mode, write experiment header
	if sp.experiment {
		sp.trace.Printf("// EXPERIMENT RESULTS\n")
		sp.trace.Printf("RunSecs, MaxHell, NegLogMaxHell, MaxJS, NegLogMaxJS, CollapseCount\n")
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
			PanicIf(ch.AdvanceChain(&wg))
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

		// Status update (including experiment file)
		if now.After(nextStatus) || !keepWorking || sp.experiment {
			runTime := time.Since(startTime).Seconds()

			if now.After(nextStatus) || !keepWorking {
				sp.mon.RunTime.Set(runTime)
				sp.out.Printf("  Samps: %12d | RT %12.2fsec\n", sampleCount, runTime)
			}

			if sp.solFile {
				merged, err := sampler.MergeChains(chains)
				if err != nil {
					return errors.Wrapf(err, "Could not merge chains to calculate score")
				}
				score, err := sol.Error(merged)
				if err != nil {
					return errors.Wrapf(err, "Error calculating score")
				}

				if now.After(nextStatus) || !keepWorking {
					errorReport(sp, "", score, true, nil)
				}

				if sp.experiment {
					colCount := 0
					for _, v := range merged {
						if v.Collapsed {
							colCount++
						}
					}
					sp.trace.Printf("%.1f, %.8f, %.5f, %.8f, %.5f, %d\n",
						runTime,
						score.MaxHellinger, -math.Log2(score.MaxHellinger),
						score.MaxJSDiverge, -math.Log2(score.MaxJSDiverge),
						colCount,
					)
				}
			}

			if now.After(nextStatus) || !keepWorking {
				nextStatus = now.Add(untilStatus)
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
			preCount := len(chains)
			chains, err = adapt.Adapt(chains, int(sp.chainAdds))
			if err != nil {
				return err
			}
			postCount := len(chains)

			if postCount != preCount {
				sp.mon.TotalChains.Set(int64(postCount))
				sp.out.Printf("ADAPT: %d Chains (was %d)\n", postCount, preCount)
			}
		}
	}

	// COMPLETED! grab results and normalize our marginals
	runTime := time.Since(startTime).Seconds()
	finalVars, err := sampler.MergeChains(chains)
	if err != nil {
		return errors.Wrapf(err, "Error in final chain merge")
	}
	for _, v := range finalVars {
		PanicIf(v.NormMarginal())
	}

	// Output the marginals we found and our final evaluation
	sp.out.Printf("DONE\n")

	// Write score if we have a solution file (and include Merlin info)
	var merlin *model.Solution
	if sp.solFile {
		score, err := sol.Error(finalVars)
		if err != nil {
			return errors.Wrapf(err, "Error calculating Final Score!")
		}
		errorReport(sp, "FINAL", score, false, nil)
		if sp.experiment {
			colCount := 0
			for _, v := range finalVars {
				if v.Collapsed {
					colCount++
				}
			}
			sp.trace.Printf("%.1f, %.8f, %.5f, %.8f, %.5f, %d\n",
				runTime,
				score.MaxHellinger, -math.Log2(score.MaxHellinger),
				score.MaxJSDiverge, -math.Log2(score.MaxJSDiverge),
				colCount,
			)

			sp.trace.Printf("// FINAL STATUS\n")
			errorReport(sp, "FINAL", score, false, sp.trace)
		}

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
			var re error
			merlin, re = model.NewSolutionFromFile(reader, merlinFilename)
			if re != nil {
				return errors.Wrapf(re, "Found merlin MAR file but could not read it")
			}

			//merlinError, re := merlin.Error(sol.Vars)
			merlinError, re := sol.Error(merlin.Vars)
			if re != nil {
				return errors.Wrapf(re, "Error calculating merlin error")
			}
			errorReport(sp, "MERLIN SCORE", merlinError, false, sp.out)
			if sp.experiment {
				sp.trace.Printf("// MERLIN SCORES\n")
				errorReport(sp, "MERLIN SCORE", merlinError, false, sp.trace)
			}

			merlinError, re = merlin.Error(finalVars)
			if re != nil {
				return errors.Wrapf(re, "Error calculating merlin error")
			}
			errorReport(sp, "OUR SCORE USING MERLIN AS SOLUTION", merlinError, false, sp.out)
		}
	}

	// Get final convergence scores (and individual errors)
	hellConverge, err := sampler.ChainConvergence(chains, model.HellingerDiff, finalVars)
	if err != nil {
		return errors.Wrapf(err, "Error getting final Hellinger Convergence")
	}
	jsConverge, err := sampler.ChainConvergence(chains, model.JSDivergence, finalVars)
	if err != nil {
		return errors.Wrapf(err, "Error getting final JS Convergence")
	}
	maxaeConverge, err := sampler.ChainConvergence(chains, model.MaxAbsDiff, finalVars)
	if err != nil {
		return errors.Wrapf(err, "Error getting final MaxAbsDiff Convergence")
	}
	avgaeConverge, err := sampler.ChainConvergence(chains, model.MeanAbsDiff, finalVars)
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
			PanicIf(sp.traceJ.Encode(v))
			sp.verb.Printf("Variable[%d] %s (Card:%d, %+v) EVID=%d\n", v.ID, v.Name, v.Card, v.State, v.FixedVal)
		}
	}
	sp.trace.Printf("// VARS (ESTIMATED)")
	for _, v := range finalVars {
		if v.FixedVal < 0 {
			PanicIf(sp.traceJ.Encode(v))
			sp.verb.Printf("Variable[%d] %s (Card:%d, %+v) %+v\n", v.ID, v.Name, v.Card, v.State, v.Marginal)
		}
	}

	// Add a variable breakdown by Merlin results if possible
	if merlin != nil {
		report := make([]*model.Variable, 0, len(finalVars))
		for i, v := range finalVars {
			if v.FixedVal >= 0 {
				continue
			}
			mv := merlin.Vars[i]
			v.State["MerlinHellError"] = model.HellingerDiff(v, mv)
			report = append(report, v)
		}

		sort.Slice(report, func(lhs, rhs int) bool {
			return report[lhs].State["MerlinHellError"] < report[rhs].State["MerlinHellError"]
		})

		sp.trace.Printf("// VARS SORTED BY DIST FROM HELLINGER")
		for _, v := range report {
			PanicIf(sp.traceJ.Encode(v))
			sp.verb.Printf("Variable[%d] %s (Card:%d, %+v) %+v\n", v.ID, v.Name, v.Card, v.State, v.Marginal)
		}
	}

	sp.trace.Printf("// OPERATING PARAMS\n")
	sp.Trace()

	sp.trace.Printf("// ENTIRE MODEL\n")
	sp.traceJ.SetIndent("", "  ")
	PanicIf(sp.traceJ.Encode(mod))

	return nil
}
