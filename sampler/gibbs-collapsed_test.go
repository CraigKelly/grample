package sampler

import (
	"fmt"
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"

	"github.com/stretchr/testify/assert"
)

// Test that we can actually sample from a simple 1-var dist
func TestWorkingGibbsCollapsed(t *testing.T) {
	assert := assert.New(t)

	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/deterministic.uai", false)
	assert.NoError(err)

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)

	samp, err := NewGibbsCollapsed(gen, mod.Clone())
	assert.NoError(err)
	assert.False(samp.baseSampler.pgm.Vars[0].Collapsed)
	assert.False(samp.baseSampler.pgm.Vars[1].Collapsed)
	assert.False(samp.baseSampler.pgm.Vars[2].Collapsed)

	for i := range mod.Vars {
		samp2, err := NewGibbsCollapsed(gen, mod.Clone())
		assert.NoError(err)
		v, err := samp2.Collapse(i)
		assert.NoError(err)
		assert.Equal(i, v.ID)
		for j, vCheck := range samp2.baseSampler.pgm.Vars {
			if j != i {
				assert.False(vCheck.Collapsed)
			} else {
				assert.True(vCheck.Collapsed)
			}
		}

		fmt.Printf("Collapsed Marginal v[%d]%v: %+v\n", i, v.Name, v.Marginal)
		assert.InEpsilon(0.50, v.Marginal[0], 1e-5)
		assert.InEpsilon(0.50, v.Marginal[1], 1e-5)
	}
}

// Test that we can actually sample from a simple 1-var dist
func TestFullGibbsCollapsed(t *testing.T) {
	assert := assert.New(t)

	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/sample.uai", false)
	assert.NoError(err)

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)

	samp, err := NewGibbsCollapsed(gen, mod.Clone())
	assert.NoError(err)
	assert.False(samp.baseSampler.pgm.Vars[0].Collapsed)
	assert.False(samp.baseSampler.pgm.Vars[1].Collapsed)

	var v *model.Variable

	v, err = samp.Collapse(0)
	assert.NoError(err)
	assert.True(v.Collapsed)
	assert.True(samp.baseSampler.pgm.Vars[0].Collapsed)
	assert.False(samp.baseSampler.pgm.Vars[1].Collapsed)

	v, err = samp.Collapse(1)
	assert.NoError(err)
	assert.True(v.Collapsed)
	assert.True(samp.baseSampler.pgm.Vars[0].Collapsed)
	assert.True(samp.baseSampler.pgm.Vars[1].Collapsed)

	// Redo model and collapse randomly
	samp, err = NewGibbsCollapsed(gen, mod)
	assert.NoError(err)

	collCount := func() int {
		c := 0
		for _, v := range samp.baseSampler.pgm.Vars {
			if v.Collapsed {
				c++
			}
		}
		return c
	}

	assert.Equal(0, collCount())

	v, err = samp.Collapse(-1)
	assert.NoError(err)
	assert.True(v.Collapsed)
	assert.Equal(1, collCount())

	v, err = samp.Collapse(-1)
	assert.NoError(err)
	assert.True(v.Collapsed)
	assert.Equal(2, collCount())

	// Our current checking requires that at least one variable remain uncollapsed
	v, err = samp.Collapse(-1)
	assert.Error(err)
	assert.Nil(v)
	assert.Equal(2, collCount())
}

var colModIts int

func runColBench(b *testing.B, m *model.Model) {
	gen, err := rand.NewGenerator(42)
	if err != nil {
		b.Fatalf("Could not init PRNG %v", err)
	}

	samp, err := NewGibbsCollapsed(gen, m)
	if err != nil {
		b.Fatalf("Could not create Gibbs-Simple sampler %v", err)
	}
	// Collapse about half the variables if possible
	for i := 0; i < len(m.Vars)/2; i++ {
		_, err := samp.Collapse(-1)
		if err != nil {
			b.Fatalf("Could not collapse var on try %d. Error: %v", i, err)
		}
	}

	oneSample := make([]int, len(m.Vars))

	b.ResetTimer()

	it := 0
	for i := 0; i < b.N; i++ {
		_, err := samp.Sample(oneSample)
		if err != nil {
			b.Fatalf("Failure on single sample (it %d) %v", i, err)
		}
		it++
	}
	colModIts = it
}

func BenchmarkGibbsCollapsedNoEvidence(b *testing.B) {
	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/Promedus_11.uai", false)
	if err != nil {
		b.Fatalf("Could not read rel 1 model %v", err)
	}

	runColBench(b, mod)
}

func BenchmarkGibbsCollapsedWithEvidence(b *testing.B) {
	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/Promedus_11.uai", true)
	if err != nil {
		b.Fatalf("Could not read rel 1 model %v", err)
	}

	runColBench(b, mod)
}
