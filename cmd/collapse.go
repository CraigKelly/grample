package cmd

import (
	"github.com/pkg/errors"

	"github.com/CraigKelly/grample/model"
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

	// TODO: actually do work
	return nil
}
