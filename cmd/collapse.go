package cmd

import (
	"math"
	"os"

	"github.com/pkg/errors"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
	"github.com/CraigKelly/grample/sampler"
)

// CollapsedIteration is a testing mode command that will iterate a model,
// collapse a single variable, and print the marginal, solution marginal, and
// error.
func CollapsedIteration(sp *startupParams) error {
	var mod *model.Model
	var sol *model.Solution
	var err error

	// We do this a lot, so create a little helper for write error metrics
	oneErrorLog := func(v1 *model.Variable, v2 *model.Variable, prefix string) error {
		score, err := model.NewErrorSuite(
			[]*model.Variable{v1},
			[]*model.Variable{v2},
		)
		if err != nil {
			return err
		}
		sp.out.Printf(
			"%s NLog | MeanAE:%7.3f MaxAE:%7.3f Hel:%7.3f JSD:%7.3f\n",
			prefix,
			-math.Log2(score.MaxMeanAbsError),
			-math.Log2(score.MaxMaxAbsError),
			-math.Log2(score.MaxHellinger),
			-math.Log2(score.MaxJSDiverge),
		)
		return nil
	}

	// Read model from file
	sp.out.Printf("Reading model from %s\n", sp.uaiFile)
	reader := model.UAIReader{}
	mod, err = model.NewModelFromFile(reader, sp.uaiFile, sp.useEvidence)
	if err != nil {
		return err
	}
	sp.out.Printf("Model has %d vars and %d functions\n", len(mod.Vars), len(mod.Funcs))

	if !sp.solFile {
		return errors.New("Itertive collapse check only works with a solution file")
	}

	solFilename := sp.uaiFile + ".MAR"
	sol, err = model.NewSolutionFromFile(reader, solFilename)
	if err != nil {
		return errors.Wrapf(err, "Could not read solution file %s", solFilename)
	}

	score, err := sol.Error(mod.Vars)
	if err != nil {
		return errors.Wrapf(err, "Error calculating init score on startup")
	}
	errorReport(sp, "ASSUME ALL MARGINALS ARE UNIFORM", score, false, sp.out)

	merlinFilename := sp.uaiFile + ".merlin.MAR"
	var merlin *model.Solution
	if _, err := os.Stat(merlinFilename); !os.IsNotExist(err) {
		merlin, err = model.NewSolutionFromFile(reader, merlinFilename)
		if err != nil {
			return errors.Wrapf(err, "Found merlin MAR file but could not read it")
		}
	}

	var merlinError *model.ErrorSuite
	if merlin != nil {
		merlinError, err = merlin.Error(mod.Vars)
		if err != nil {
			return errors.Wrapf(err, "Error calculating merlin error on startup")
		}
		errorReport(sp, "MERLIN SCORE", merlinError, false, sp.out)
	}

	gen, err := rand.NewGenerator(sp.randomSeed)
	if err != nil {
		return err
	}

	for i, v := range mod.Vars {
		sp.out.Printf("--------------------------------------------------\n")
		sp.out.Printf("Check for Var[%v] %v\n", v.ID, v.Name)

		if v.FixedVal >= 0 {
			sp.out.Printf("Skipping: has FixedVal=%d\n", v.FixedVal)
			continue
		}

		samp, err := sampler.NewGibbsCollapsed(gen, mod.Clone())
		if err != nil {
			return errors.Wrapf(err, "Sampler fail on var %+v", v)
		}

		blanketSize := samp.BlanketSize(v)
		sp.out.Printf("BlanketSize: %d, FuncCount: %d\n", blanketSize, samp.FunctionCount(v))
		if blanketSize > sampler.NeighborVarMax {
			sp.out.Printf("SKIPPING: BlanketSize %d > %d\n", blanketSize, sampler.NeighborVarMax)
			continue
		}

		solVar := sol.Vars[i]
		colVar, err := samp.Collapse(i)
		if err != nil {
			return err
		}
		if solVar.ID != colVar.ID {
			return errors.Errorf("Solution/Model var mismatch %v != %v", solVar.ID, colVar.ID)
		}

		sp.out.Printf("COLLAPSED: %8.5f\n", colVar.Marginal)
		sp.out.Printf("SOLUTION : %8.5f\n", solVar.Marginal)

		err = oneErrorLog(colVar, solVar, "Col Vs Sol")
		if err != nil {
			return err
		}

		if merlin != nil {
			merVar := merlin.Vars[i]
			if merVar.ID != colVar.ID {
				return errors.Errorf("Merlin var mismatch")
			}

			sp.out.Printf("MERLIN   : %8.5f\n", merVar.Marginal)

			err = oneErrorLog(merVar, solVar, "Mer vs Sol")
			if err != nil {
				return err
			}

			err = oneErrorLog(merVar, colVar, "Mer vs COL")
			if err != nil {
				return err
			}
		}
	}

	sp.out.Printf("--------------------------------------------------\n")

	return nil
}
