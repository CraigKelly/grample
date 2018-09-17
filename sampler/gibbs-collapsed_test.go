package sampler

import (
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"

	"github.com/stretchr/testify/assert"
)

// Test that we can actually sample from a simple 1-var dist
func TestWorkingGibbsCollapsed(t *testing.T) {
	assert := assert.New(t)

	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/one.uai", false)
	assert.NoError(err)

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)

	samp, err := NewGibbsCollapsed(gen, mod)
	assert.NoError(err)
	assert.False(samp.baseSampler.pgm.Vars[0].Collapsed)

	oneSample := make([]int, 1)
	counts := make([]int, 2)
	for i := 0; i < 4096; i++ {
		idx, err := samp.Sample(oneSample)
		assert.Equal(0, idx)
		assert.NoError(err)
		if err != nil {
			break
		}
		counts[oneSample[0]]++
	}

	// Technically just highly unlikely...
	assert.True(counts[0] > 0)
	assert.True(counts[1] > 0)

	pgm := samp.baseSampler.pgm

	v := pgm.Vars[0]
	v.Marginal[0] = float64(counts[0]) / 4096.0
	v.Marginal[1] = float64(counts[1]) / 4096.0
	assert.InEpsilon(0.25, pgm.Vars[0].Marginal[0], 0.2)
	assert.InEpsilon(0.75, pgm.Vars[0].Marginal[1], 0.2)

	v, err = samp.Collapse(0)
	assert.Equal(0, v.ID)
	assert.NoError(err)

	// Keep this in once we're fixed
	assert.InEpsilon(0.25, pgm.Vars[0].Marginal[0], 1e-5)
	assert.InEpsilon(0.75, pgm.Vars[0].Marginal[1], 1e-5)
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

	v, err = samp.Collapse(-1)
	assert.NoError(err)
	assert.True(v.Collapsed)
	assert.Equal(3, collCount())

	v, err = samp.Collapse(-1)
	assert.Error(err)
	assert.Nil(v)
	assert.Equal(3, collCount())

	// TODO: check collapsed vars
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
