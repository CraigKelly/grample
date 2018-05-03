package model

import (
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// TODO: test

// UAIReader reads the UAI inference data set format. This format has also been
// used at competitions like PIC2001 at PASCAL2. In fact, a very good
// description of the format is available at
// http://www.cs.huji.ac.il/project/PASCAL/fileFormat.php
type UAIReader struct {
}

// ReadModel implements the model.Reader interface
func (r UAIReader) ReadModel(data []byte) (*Model, error) {
	// We counted: bayes net with single var with card=1 with minimal spacing
	// takes up 15 chars
	if len(data) < 15 {
		return nil, errors.Errorf("Invalid data buffer: len=%d (<15)", len(data))
	}

	// A minimal model will have 6 fields
	fr := newFieldReader(data)
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
	// TODO

	return m, nil
}

// fieldReader is just a simple reader for basic file formats.
// TODO: extract this somewhere when we can read more file formats.
type fieldReader struct {
	Pos    int
	Fields []string
}

func newFieldReader(data []byte) *fieldReader {
	return &fieldReader{0, strings.Fields(string(data))}
}

func (fr *fieldReader) Read() (string, error) {
	if fr.Pos >= len(fr.Fields) {
		return "", io.EOF
	}
	p := fr.Pos
	fr.Pos++
	return fr.Fields[p], nil
}

func (fr *fieldReader) ReadInt() (int, error) {
	s, err := fr.Read()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 0)
	return int(i), err
}

func (fr *fieldReader) ReadFloat() (float64, error) {
	s, err := fr.Read()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}
