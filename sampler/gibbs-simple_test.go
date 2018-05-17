package sampler

import (
	"testing"

	"github.com/CraigKelly/grample/model"
	"github.com/CraigKelly/grample/rand"
)

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
