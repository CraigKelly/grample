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

// Reader implementors instantiate a model from a byte stream
type Reader interface {
	ReadModel(data []byte) (*Model, error)
}

// TODO: Evidence reader?

// Model represent a PGM
type Model struct {
	Type  string      // PGM type - should match a constant
	Name  string      // Model name
	Vars  []*Variable // Variables (nodes) in the model
	Funcs []*Function // Function of variables (CPT) in the model
}

// NewModelFromFile initializes and creates a model from the specified source.
func NewModelFromFile(r Reader, filename string) (*Model, error) {
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

// Check returns an error if there is a problem with the model
func (m Model) Check() error {
	if m.Type != BAYES && m.Type != MARKOV {
		return errors.Errorf("Unknown model type %s", m.Type)
	}

	for _, v := range m.Vars {
		e := v.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Variable %s", m.Name, v.Name)
		}
	}

	for _, f := range m.Funcs {
		e := f.Check()
		if e != nil {
			return errors.Wrapf(e, "Model %s has an invalid Function %s", m.Name, f.Name)
		}
	}

	return nil
}
