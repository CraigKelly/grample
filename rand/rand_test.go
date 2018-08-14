package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMTBadSeed(t *testing.T) {
	assert := assert.New(t)

	gen, err := NewGeneratorSlice([]uint64{})
	assert.Nil(gen)
	assert.Error(err)
}

func TestMTCanonicalSeed(t *testing.T) {
	assert := assert.New(t)

	gen, err := NewGeneratorSlice([]uint64{0x12345, 0x23456, 0x34567, 0x45678})
	assert.NotNil(gen)
	assert.NoError(err)

	origTestSeq := []uint64{
		7266447313870364031,
		4946485549665804864,
		16945909448695747420,
		16394063075524226720,
		4873882236456199058,
	}

	// Now convert to the format we should get from Int63
	for _, v := range origTestSeq {
		exp := int64(v & 0x7fffffffffffffff)
		act := gen.Int63()
		assert.Equal(exp, act)
		// fmt.Printf("%v %v => %v\n", exp, act, exp-act)
	}
}
