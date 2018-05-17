package sampler

import (
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"

	"github.com/stretchr/testify/assert"
)

// Test that we can actually sample from a simple 1-var dist
func TestWorkingGibbsSimple(t *testing.T) {
	assert := assert.New(t)

	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/one.uai")
	assert.NoError(err)

	gen, err := rand.NewGenerator(42)
	assert.NoError(err)

	samp, err := NewGibbsSimple(gen, mod)
	assert.NoError(err)

	oneSample := make([]int, 1)
	counts := make([]int, 2)
	for i := 0; i < 1024; i++ {
		err = samp.Sample(oneSample)
		assert.NoError(err)
		counts[oneSample[0]]++
	}

	// Technically just highly unlikely...
	assert.True(counts[0] > 0)
	assert.True(counts[1] > 0)
}

var modIts int

func BenchmarkGibbsSimple(b *testing.B) {
	var err error

	reader := model.UAIReader{}
	mod, err := model.NewModelFromFile(reader, "../res/relational_1.uai")
	if err != nil {
		b.Fatalf("Could not read rel 1 model %v", err)
	}

	gen, err := rand.NewGenerator(42)
	if err != nil {
		b.Fatalf("Could not init PRNG %v", err)
	}

	samp, err := NewGibbsSimple(gen, mod)
	if err != nil {
		b.Fatalf("Could not create Gibbs-Simple sampler %v", err)
	}

	oneSample := make([]int, len(mod.Vars))

	b.ResetTimer()

	it := 0
	for i := 0; i < b.N; i++ {
		err = samp.Sample(oneSample)
		if err != nil {
			b.Fatalf("Failure on single sample (it %d) %v", i, err)
		}
		it++
	}
	modIts = it
}
