package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorld_Regression1(t *testing.T) {
	playthrough := DeserializePlaythrough(ReadFile("average-playthrough.clone1"))
	expected := string(ReadFile("average-playthrough.clone1-hash"))
	actual := RegressionId(&playthrough)
	println(actual)
	assert.Equal(t, expected, actual)
}

// Playthrough with 5899 frames.
// BenchmarkAveragePlaythrough-12    	      10	 104547620 ns/op
func BenchmarkAveragePlaythrough(b *testing.B) {
	playthrough := DeserializePlaythrough(ReadFile("average-playthrough.clone1"))
	println(len(playthrough.History))
	for b.Loop() {
		world := NewWorld()
		for i := range len(playthrough.History) {
			world.Step(playthrough.History[i])
		}
	}
}
