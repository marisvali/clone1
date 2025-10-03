package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRand_SameSeedSameRandomNumbers(t *testing.T) {
	r1 := NewRand(13)
	v1 := [10]int64{}
	for i := range v1 {
		v1[i] = r1.RInt(0, 1000000)
	}

	r2 := NewRand(13)
	v2 := [10]int64{}
	for i := range v2 {
		v2[i] = r2.RInt(0, 1000000)
	}

	assert.Equal(t, v1, v2)
}

func TestRand_DifferentSeedsDifferentRandomNumbers(t *testing.T) {
	r1 := NewRand(13)
	v1 := [10]int64{}
	for i := range v1 {
		v1[i] = r1.RInt(0, 1000000)
	}

	r2 := NewRand(14)
	v2 := [10]int64{}
	for i := range v2 {
		v2[i] = r2.RInt(0, 1000000)
	}

	assert.NotEqual(t, v1, v2)
}

func TestRand_CopyMakesIdenticalGenerators(t *testing.T) {
	r1 := NewRand(13)
	vOriginal := [10]int64{}
	for i := range vOriginal {
		vOriginal[i] = r1.RInt(0, 1000000)
	}

	r2 := r1

	v1 := [10]int64{}
	for i := range v1 {
		v1[i] = r1.RInt(0, 1000000)
	}

	v2 := [10]int64{}
	for i := range v2 {
		v2[i] = r2.RInt(0, 1000000)
	}

	assert.Equal(t, v1, v2)

	for i := range v1 {
		v1[i] = r1.RInt(0, 1000000)
		v2[i] = r2.RInt(0, 1000000)
	}

	assert.Equal(t, v1, v2)
}
