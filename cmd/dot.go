package cmd

import (
	"log"

	"github.com/pkg/errors"

	"github.com/CraigKelly/grample/model"
)

// TODO: display factors somehow (dot cluster func? factor graph?)
// TODO: include cardinality in nodes
// TODO: optionally show evidence in nodes
// TODO: optionally show MAR from solution in nodes
// TODO: optionally use merlin solution

// DotOutput reads a given model and outputs a graphviz description
func DotOutput(sp *startupParams) error {
	var mod *model.Model
	var err error

	// Read model from file
	sp.out.Printf("Reading model from %s\n", sp.uaiFile)
	reader := model.UAIReader{}
	mod, err = model.NewModelFromFile(reader, sp.uaiFile, sp.useEvidence)
	if err != nil {
		return err
	}
	sp.out.Printf("Model has %d vars and %d functions\n", len(mod.Vars), len(mod.Funcs))

	// Find all variable linkages
	type AdjMap map[int]bool
	varAdj := make(map[int]AdjMap)

	for i, v := range mod.Vars {
		if i != v.ID {
			return errors.Errorf("Var %v has ID %d != idx %d", v.Name, v.ID, i)
		}
		varAdj[v.ID] = make(AdjMap)
	}

	for _, f := range mod.Funcs {
		for i, v1 := range f.Vars {
			for _, v2 := range f.Vars[i+1:] {
				varAdj[v1.ID][v2.ID] = true
			}
		}
	}

	var target *log.Logger
	if len(sp.traceFile) > 0 {
		sp.out.Printf("Writing model to trace file %v\n", sp.traceFile)
		target = sp.trace
	} else {
		target = sp.out
	}

	// Start graph
	target.Printf("strict graph G {\n")

	// Output vars
	//for _, v := range mod.Vars {
	//	target.Printf("    node %s\n", v.Name)
	//}

	// Output links
	for _, v1 := range mod.Vars {
		adj := varAdj[v1.ID]
		for v2id := range adj {
			v2 := mod.Vars[v2id]
			target.Printf("    %s -- %s;\n", v1.Name, v2.Name)
		}
	}

	// Finish graph
	target.Printf("}\n")

	return nil
}
