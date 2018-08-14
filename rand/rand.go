package rand

import (
	"github.com/pkg/errors"
	"github.com/seehuhn/mt19937"
)

// A Generator uses a goroutine to populate batches of random numbers. One day
// is will also use a better PRNG, like the Mersenne twister.
type Generator struct {
	ch chan int64
}

// NewGeneratorSlice starts a new background PRNG based on the given seed
// slice. If the slice has only one entry, then the MT generator is
// initialized with Seed. Otherwise SeedFromSlice is used
func NewGeneratorSlice(seed []uint64) (*Generator, error) {
	if len(seed) < 1 {
		return nil, errors.Errorf("Invalid generator seed array %v", seed)
	}

	numChan := make(chan int64, 1024)

	r := mt19937.New()
	if len(seed) == 1 {
		r.Seed(int64(seed[0]))
	} else {
		r.SeedFromSlice(seed)
	}

	go func() {
		for {
			numChan <- r.Int63()
		}
	}()

	g := &Generator{
		ch: numChan,
	}

	return g, nil
}

// NewGenerator is a helper wrapper around NewGeneratorSlice
func NewGenerator(seed int64) (*Generator, error) {
	return NewGeneratorSlice([]uint64{uint64(seed)})
}

// Int63 provides the same interface as Go's math/rand, but with pre-generation.
func (g *Generator) Int63() int64 {
	return <-g.ch
}

// Int63n is a copy of the current Go code
func (g *Generator) Int63n(n int64) int64 {
	if n <= 0 {
		panic("invalid argument to Int63n")
	}

	if n&(n-1) == 0 { // n is power of two, can mask
		return g.Int63() & (n - 1)
	}

	max := int64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := g.Int63()
	for v > max {
		v = g.Int63()
	}

	return v % n
}

// Int31 is just a copy of the golang impl
func (g *Generator) Int31() int32 {
	return int32(g.Int63() >> 32)
}

// Int31n is just a copy of the golang impL
func (g *Generator) Int31n(n int32) int32 {
	if n <= 0 {
		panic("invalid argument to Int31n")
	}

	if n&(n-1) == 0 { // n is power of two, can mask
		return g.Int31() & (n - 1)
	}

	max := int32((1 << 31) - 1 - (1<<31)%uint32(n))
	v := g.Int31()

	for v > max {
		v = g.Int31()
	}

	return v % n
}

// Float64 uses the commented, simpler implmentation since we don't have the
// same support requirements for users
func (g *Generator) Float64() float64 {
	// See the Go lang comments for Rand Float64 implementation for details
	return float64(g.Int63n(1<<53)) / (1 << 53)
}
