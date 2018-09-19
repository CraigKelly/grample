package cmd

import (
	"math"

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
	errorReport(sp, "ASSUME ALL MARGINALS ARE UNIFORM", score, false)

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

		score, err := model.NewErrorSuite(
			[]*model.Variable{solVar},
			[]*model.Variable{colVar},
		)
		if err != nil {
			return err
		}

		sp.out.Printf(
			"NLog | MeanAE:%.3f MaxAE:%.3f Hel:%.3f JSD:%.3f\n",
			-math.Log2(score.MaxMeanAbsError),
			-math.Log2(score.MaxMaxAbsError),
			-math.Log2(score.MaxHellinger),
			-math.Log2(score.MaxJSDiverge),
		)
	}

	sp.out.Printf("--------------------------------------------------\n")

	return nil
}
