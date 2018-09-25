package model

import (
	"strings"

	"github.com/pkg/errors"
)

// UAIReader reads the UAI inference data set format. This format has also been
// used at competitions like PIC2001 at PASCAL2. In fact, a very good
// description of the format is available at
// http://www.cs.huji.ac.il/project/PASCAL/fileFormat.php
type UAIReader struct {
}

// Preprocessor for UAI files: remove lines that are blank or comments. Return
// the new buffer and the count of "real" lines found. If reqPrefix is
// specified, then a line starting with reqPrefix must be present AND all text
// before the first occurrence of reqPrefix will be dropped.
func uaiPreprocess(data []byte, reqPrefix string) (string, int) {
	lines := strings.Split(string(data), "\n")

	startFound := false
	if len(reqPrefix) < 1 {
		startFound = true // No req prefix specified
	}

	newPos := 0
	for i, ln := range lines {
		ln = strings.TrimSpace(ln)
		if len(ln) < 1 || ln[0] == 'c' {
			lines[i] = "" // Empty or comment: skip
			continue
		}

		if !startFound {
			if strings.HasPrefix(ln, reqPrefix) {
				startFound = true
			} else {
				continue // still looking...
			}
		}

		// Rewrite update line and update insert point
		lines[newPos] = ln
		newPos++
	}

	return strings.Join(lines[:newPos], "\n"), newPos
}

// ReadModel implements the model.Reader interface
func (r UAIReader) ReadModel(data []byte) (*Model, error) {
	// We counted: bayes net with single var with card=1 with minimal spacing
	// takes up 15 chars
	if len(data) < 15 {
		return nil, errors.Errorf("Invalid data buffer: len=%d (<15)", len(data))
	}

	// A minimal model will have 6 fields
	text, lineCount := uaiPreprocess(data, "")
	if lineCount < 1 {
		return nil, errors.Errorf("No lines found in file")
	}
	fr := NewFieldReader(text)
	if len(fr.Fields) < 6 {
		return nil, errors.Errorf("Invalid data: only %d fields found (<6)", len(fr.Fields))
	}

	// Network type
	m := &Model{}

	var err error
	m.Type, err = fr.Read()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading UAI file on Type")
	}
	if m.Type != BAYES && m.Type != MARKOV {
		return nil, errors.Errorf("Unknown model type %v", m.Type)
	}

	// Network variables: count followed by cardinality.  For example, 3 boolean
	// variables would be "3 2 2 2"
	var varCount int
	varCount, err = fr.ReadInt()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading UAI file on Variable count")
	}
	if varCount < 1 {
		return nil, errors.Errorf("Invalid variable count: %d", varCount)
	}

	m.Vars = make([]*Variable, varCount)
	var card int
	for i := 0; i < varCount; i++ {
		card, err = fr.ReadInt()
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading Card for var %d", i)
		}
		if card < 1 {
			return nil, errors.Errorf("Invalid card %d for var %d", card, i)
		}

		m.Vars[i], err = NewVariable(i, card)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create variable from UAI file")
		}
	}

	// Network cliques and factor (make up the functions)
	// First we read the clique/function count
	var funcCount int
	funcCount, err = fr.ReadInt()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading UAI file on Clique count")
	}
	if funcCount < 1 {
		return nil, errors.Errorf("Invalid Clique count count: %d", funcCount)
	}

	// Then we read the variables (domain) for the functions - they are count
	// followed by var indexes.  For a model variables [A,B,C], a function over
	// [B,C] would have the line "2 1 2". A function over all three variables
	// would have "3 0 1 2".
	m.Funcs = make([]*Function, funcCount)
	for i := 0; i < funcCount; i++ {
		varCount, err = fr.ReadInt()
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading Clique size for Clique %d", i)
		}
		if varCount < 1 {
			return nil, errors.Errorf("Invalid variable count (<1) for Clique %d", i)
		}

		fvars := make([]*Variable, varCount)
		for j := 0; j < varCount; j++ {
			varIdx, err := fr.ReadInt()
			if err != nil {
				return nil, errors.Wrapf(err, "Error reading var idx for Clique %d Variable %d", i, j)
			}
			if varIdx < 0 || varIdx >= len(m.Vars) {
				return nil, errors.Errorf("Invalid var idx %d for Clique %d Variable %d", varIdx, i, j)
			}

			fvars[j] = m.Vars[varIdx]
		}

		m.Funcs[i], err = NewFunction(i, fvars)
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating function %d", i)
		}
	}

	// Now we read in the table that NewFunction initialized. The order of
	// Function.Table in designed to match the order in a UAI file, so this
	// will straightforward
	var tabSize int
	var entry float64
	for _, fun := range m.Funcs {
		tabSize, err = fr.ReadInt()
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading table size on function %s", fun.Name)
		}
		if tabSize != len(fun.Table) {
			return nil, errors.Errorf("Read table size %d != previous Clique size %d on function %s", tabSize, len(fun.Table), fun.Name)
		}

		for t := 0; t < tabSize; t++ {
			entry, err = fr.ReadFloat()
			if err != nil {
				return nil, errors.Errorf("Error reading entry %d on function %s", t, fun.Name)
			}
			fun.Table[t] = entry
		}
	}

	// Finally all done - we leave it to our caller to perform final checking
	return m, nil
}

// ApplyEvidence is part of the reader interface - read the evidence file and
// apply to the model.
func (r UAIReader) ApplyEvidence(data []byte, m *Model) error {
	text, lineCount := uaiPreprocess(data, "")
	if lineCount < 1 {
		return errors.Errorf("Invalid data buffer: there is no data")
	} else if lineCount > 2 {
		return errors.Errorf("Found %d lines: only understand evidence files with 1 or 2 lines", lineCount)
	}

	fr := NewFieldReader(text)
	if len(fr.Fields) < 1 {
		return errors.Errorf("Invalid data: found no fields")
	}

	var err error

	sampleCount := 1 // default to 1 sample (1-line evidence file format)
	if lineCount == 2 {
		sampleCount, err = fr.ReadInt()
		if err != nil {
			return errors.Wrapf(err, "Error reading UAI evid file sample count")
		}
		if sampleCount == 0 {
			return nil // Allowed
		}
		if sampleCount > 1 {
			return errors.Errorf("Sample count is %d - only single sample evidence currently supported", sampleCount)
		}
	}

	// Read variable count
	var varCount int
	varCount, err = fr.ReadInt()
	if err != nil {
		return errors.Wrap(err, "Error reading UAI evid Variable Count")
	}
	if varCount < 1 {
		return nil // Allowed
	}

	var idx int
	var val int
	for i := 0; i < varCount; i++ {
		idx, err = fr.ReadInt()
		if err != nil {
			return errors.Wrapf(err, "Could not read evid var on iteration %d", i)
		}
		if idx < 0 || idx >= len(m.Vars) {
			return errors.Errorf("Read incorrect variable index %d", idx)
		}

		v := m.Vars[idx]
		if v.FixedVal != -1 {
			return errors.Errorf("variable[%d]:%v had previous fixedval %d", idx, v.Name, v.FixedVal)
		}

		val, err = fr.ReadInt()
		if err != nil {
			return errors.Wrapf(err, "Could not read evid var value on iteration %d, index %d", i, idx)
		}
		if val < 0 || val >= v.Card {
			return errors.Errorf("Read invalid value %d for variable[%d]:%v with card %d", val, idx, v.Name, v.Card)
		}

		v.FixedVal = val
	}

	return nil
}

// ReadMargSolution implements the model.SolReader interface
func (r UAIReader) ReadMargSolution(data []byte) (*Solution, error) {
	// We counted: 1 var with card 1 is MAR 1 1 1.0
	if len(data) < 11 {
		return nil, errors.Errorf("Invalid data buffer: len=%d (<11)", len(data))
	}

	// A minimal solution will have 3 fields
	// Note that we only read one MAR solution, *BUT* we'll skip anything
	// before it. This is mainly useful for Merlin MAR files because Merlin
	// includes a PR solution section before the MAR section.
	text, lineCount := uaiPreprocess(data, "MAR")
	if lineCount < 1 {
		return nil, errors.Errorf("No lines in file")
	}
	fr := NewFieldReader(text)
	if len(fr.Fields) < 4 {
		return nil, errors.Errorf("Invalid data: only %d fields found (<4)", len(fr.Fields))
	}

	var err error

	// Check solution type
	solType, err := fr.Read()
	if err != nil {
		return nil, errors.Wrap(err, "Could not understand file")
	}
	if solType != "MAR" {
		return nil, errors.Errorf("Unknown solution file type %s", solType)
	}

	// Read variable count
	var varCount int
	varCount, err = fr.ReadInt()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading UAI MAR Solution Variable Count")
	}
	if varCount < 1 {
		return nil, errors.Errorf("Invalid variable count: %d", varCount)
	}

	// Read variables and their marginals
	sol := &Solution{
		Vars: make([]*Variable, varCount),
	}

	var card int
	for i := 0; i < varCount; i++ {
		card, err = fr.ReadInt()
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading Card for var %d", i)
		}
		if card < 1 {
			return nil, errors.Errorf("Invalid card %d for var %d", card, i)
		}

		sol.Vars[i], err = NewVariable(i, card)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create variable from UAI MAR Sol file")
		}

		var p float64
		for m := 0; m < card; m++ {
			p, err = fr.ReadFloat()
			if err != nil {
				return nil, errors.Wrapf(err, "Could not read marg prob %d on var %d (%s)", m, i, sol.Vars[i].Name)
			}
			if p < 0.0 || p > 1.0 {
				return nil, errors.Wrapf(err, "Invalid p=%f marg prob %d on var %d (%s)", p, m, i, sol.Vars[i].Name)
			}
			sol.Vars[i].Marginal[m] = p
		}

		err = sol.Vars[i].NormMarginal()
		if err != nil {
			return nil, errors.Wrapf(err, "Marginal Invalid on var %d (%s)", i, sol.Vars[i].Name)
		}
	}

	// Finally all done - we leave it to our caller to perform final checking
	return sol, nil
}
