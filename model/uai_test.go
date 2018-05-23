package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const PASCALExample = `MARKOV
3
2 2 3
3
1 0
2 0 1
2 1 2

2
 0.436 0.564

4
 0.128 0.872
 0.920 0.080

6
 0.210 0.333 0.457
 0.811 0.000 0.189
`

// Test preprocessing
func TestUAIPreproc(t *testing.T) {
	assert := assert.New(t)

	assertPreproc := func(lineCount int, correct string, buf string) {
		s, c := uaiPreprocess([]byte(buf))
		assert.Equal(lineCount, c)
		assert.Equal(correct, s)
	}

	assertPreproc(0, "", "")
	assertPreproc(0, "", "\n\n\n")
	assertPreproc(0, "", "c\nc\ncnope")

	assertPreproc(1, "abc", " abc ")
	assertPreproc(1, "abc", "abc\nc comment\n")
	assertPreproc(1, "abc", "\n\n\n\nc comment\n\n\nabc")

	assertPreproc(2, "hello\nworld", "hello\nworld")
	assertPreproc(2, "hello\nworld", "hello\nworld\n")
	assertPreproc(2, "hello\nworld", "\nhello\n\nworld\n")
	assertPreproc(2, "hello\nworld", "c comment\n\nhello\nc again\nworld\nc last\n\n")
}

// Test reading the example file at http://www.cs.huji.ac.il/project/PASCAL/fileFormat.php#model
func TestUAIDoc(t *testing.T) {
	assert := assert.New(t)

	r := UAIReader{}
	m, err := NewModelFromBuffer(r, []byte(PASCALExample))
	assert.NoError(err)
	assert.NoError(m.Check())

	assert.Equal(MARKOV, m.Type)

	assert.Equal(3, len(m.Vars))
	assert.Equal(2, m.Vars[0].Card)
	assert.Equal(2, m.Vars[1].Card)
	assert.Equal(3, m.Vars[2].Card)

	assert.Equal(3, len(m.Funcs))

	cases := []struct {
		cards []int
		table []float64
	}{
		{[]int{2}, []float64{0.436, 0.564}},
		{[]int{2, 2}, []float64{0.128, 0.872, 0.920, 0.080}},
		{[]int{2, 3}, []float64{0.210, 0.333, 0.457, 0.811, 0.000, 0.189}},
	}

	for i, c := range cases {
		fun := m.Funcs[i]

		assert.Equal(len(c.cards), len(fun.Vars))
		for j, card := range c.cards {
			assert.Equal(card, fun.Vars[j].Card)
		}

		assert.Equal(len(c.table), len(fun.Table))
		for j, val := range c.table {
			assert.Equal(val, fun.Table[j])
		}
	}

	const EPS = 1e-12

	val, err := m.Funcs[2].Eval([]int{1, 2}) // last val of last function
	assert.NoError(err)
	assert.True(math.Abs(val-0.189) < EPS)
}

// Test reading a pretty large file from disk
func TestUAILargeFile(t *testing.T) {
	assert := assert.New(t)

	r := UAIReader{}
	m, err := NewModelFromFile(r, "../res/relational_1.uai", false)
	assert.NoError(err)
	assert.NoError(m.Check())

	assert.Equal(MARKOV, m.Type)

	assert.Equal(500, len(m.Vars))
	assert.Equal(62500, len(m.Funcs))

	val, err := m.Funcs[62499].Eval([]int{1, 1}) // Last val of last func
	assert.NoError(err)
	assert.InEpsilon(val, 1.00752819544, 1e-12)
}

// Test reading a solution file
func TestUAIMarSolFile(t *testing.T) {
	assert := assert.New(t)

	r := UAIReader{}
	m, err := NewModelFromFile(r, "../res/one.uai", false)
	assert.NoError(err)
	assert.NoError(m.Check())
	assert.Equal(-1, m.Vars[0].FixedVal)

	s, err := NewSolutionFromFile(r, "../res/one.uai.MAR")
	assert.NoError(err)
	assert.NoError(s.Check(m))

	// Handy to know: our simple one.uai model has a single factor of 0.25/0.75
	// and models default the vars to have uniform marginals. So we know the
	// starting TAE should be 0.5
	score, _, err := s.AbsError(m)
	assert.NoError(err)
	assert.InEpsilon(0.5, score, 1e-8)

	// Also check non-normed model vars
	m.Vars[0].Marginal[0] = 250.0
	m.Vars[0].Marginal[1] = 250.0
	score, _, err = s.AbsError(m)
	assert.NoError(err)
	assert.InEpsilon(0.5, score, 1e-8)
}

// Test reading evidence
func TestUAIMariEvidFile(t *testing.T) {
	assert := assert.New(t)

	r := UAIReader{}

	// Basic file reading
	m, err := NewModelFromFile(r, "../res/one.uai", true)
	assert.NoError(err)
	assert.NoError(m.Check())
	// Remember that the default evid file has no evidence
	assert.Equal(-1, m.Vars[0].FixedVal)

	// Check that we can't support multi-sample evidence
	err = r.ApplyEvidence([]byte("2\n1 0 0\n1 0 1"), m)
	assert.Error(err)
	assert.Equal(-1, m.Vars[0].FixedVal)

	// Test multiple formats and re-application error
	checkOneVarSet := func(evid string, expVal int) {
		m, err := NewModelFromFile(r, "../res/one.uai", false)
		assert.NoError(err)
		assert.NoError(m.Check())
		assert.Equal(-1, m.Vars[0].FixedVal)

		err = r.ApplyEvidence([]byte(evid), m)
		assert.NoError(err)
		assert.Equal(expVal, m.Vars[0].FixedVal)

		// Now check that re-applying evidence fails
		err = r.ApplyEvidence([]byte(evid), m)
		assert.Error(err)
	}

	checkOneVarSet("1 0 0", 0)
	checkOneVarSet("1\n1 0 1", 1)
}
