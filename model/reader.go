package model

import (
	"io"
	"strconv"
	"strings"
)

// FieldReader is just a simple reader for basic file formats.
type FieldReader struct {
	Pos    int
	Fields []string
}

// NewFieldReader constructs a new field reader around the given data
func NewFieldReader(data string) *FieldReader {
	return &FieldReader{0, strings.Fields(data)}
}

// Read returns the next space-delimited field/token
func (fr *FieldReader) Read() (string, error) {
	if fr.Pos >= len(fr.Fields) {
		return "", io.EOF
	}
	p := fr.Pos
	fr.Pos++
	return fr.Fields[p], nil
}

// ReadInt reads the next token as an int
func (fr *FieldReader) ReadInt() (int, error) {
	s, err := fr.Read()
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(s, 10, 0)
	return int(i), err
}

// ReadFloat reads the next token as a float
func (fr *FieldReader) ReadFloat() (float64, error) {
	s, err := fr.Read()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}
