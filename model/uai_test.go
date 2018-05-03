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

// Test reading the example file at http://www.cs.huji.ac.il/project/PASCAL/fileFormat.php#model
func TestUAIDocFile(t *testing.T) {
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
