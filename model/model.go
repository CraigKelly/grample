package model

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

// Model type constant string - matches UAI formats
const (
	BAYES  = "BAYES"
	MARKOV = "MARKOV"
)

// Reader implementors instantiate a model from a byte stream and optionally
// applies evidence from a second byte stream.
type Reader interface {
	ReadModel(data []byte) (*Model, error)
	ApplyEvidence(data []byte, m *Model) error
}

// Model represent a PGM
type Model struct {
	Type  string      // PGM type - should match a constant
	Name  string      // Model name
	Vars  []*Variable // Variables (nodes) in the model
	Funcs []*Function `json:"-"` // Function of variables (CPT) in the model
}

// Clone returns a copy of the current model. Note that marginal state will be copied as well.
func (m *Model) Clone() *Model {
	cp := &Model{
		Type:  m.Type,
		Name:  m.Name,
		Vars:  make([]*Variable, len(m.Vars)),
		Funcs: make([]*Function, len(m.Funcs)),
	}

	for i, v := range m.Vars {
		cp.Vars[i] = v.Clone()
	}

	for i, f := range m.Funcs {
		cp.Funcs[i] = f.Clone()
	}

	return cp
}

// NewModelFromFile initializes and creates a model from the specified source.
func NewModelFromFile(r Reader, filename string, useEvidence bool) (*Model, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not READ model from %s", filename)
	}

	model, err := NewModelFromBuffer(r, data)
	if err != nil {
		return nil, err
	}

	// Name the model from the file
	var ext = filepath.Ext(filename)
	model.Name = filename[0 : len(filename)-len(ext)]

	// Apply evidence if necessary
	if useEvidence {
		err = model.ApplyEvidenceFromFile(r, filename+".evid")
	}

	return model, nil
}

// NewModelFromBuffer creates a model from the given pre-read data
func NewModelFromBuffer(r Reader, data []byte) (*Model, error) {
	m, err := r.ReadModel(data)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not PARSE model")
	}

	err = m.Check()
	if err != nil {
		return nil, errors.Wrapf(err, "Parsed model is not valid")
	}

	return m, nil
}

// ApplyEvidenceFromFile will read, parse, and apply the evidence
func (m *Model) ApplyEvidenceFromFile(r Reader, eviFilename string) error {
	// We currently only support one evidence file applied, so we start by
	// resetting all var's no unfixed/no evidence
	for _, v := range m.Vars {
		v.FixedVal = -1
	}

	data, err := ioutil.ReadFile(eviFilename)
	if err != nil {
		return errors.Wrapf(err, "Could not READ model evidence from %s", eviFilename)
	}

	err = r.ApplyEvidence(data, m)
	if err != nil {
		return errors.Wrapf(err, "Could not apply evidence to model %s", m.Name)
	}

	return nil
}

// Check returns an error if there is a problem with the model
func (m *Model) Check() error {
	if m.Type != BAYES && m.Type != MARKOV {
		return errors.Errorf("Unknown model type %s", m.Type)
	}

	varID := make(map[int]bool)
	fixCount := 0
	for _, v := range m.Vars {
		e := v.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Variable %s", m.Name, v.Name)
		}

		_, ok := varID[v.ID]
		if ok {
			return errors.Errorf("Duplicate Id %d for Var %s", v.ID, v.Name)
		}
		varID[v.ID] = true

		if v.FixedVal > -1 {
			fixCount++
		}
	}
	if fixCount >= len(m.Vars) {
		return errors.Errorf("Fixed variable count is %d - all vars are fixed!", fixCount)
	}

	for _, f := range m.Funcs {
		e := f.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Function %s", m.Name, f.Name)
		}
	}

	return nil
}
