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
// Tests below performed on my ThinkPad P52, unplugged.
// before doing anything:
// BenchmarkAveragePlaythrough-12    	      10	 104547620 ns/op
// after pre-allocating slice for obstacles +4 to include top/bottom/left/right
// BenchmarkAveragePlaythrough-12    	      13	  80981146 ns/op
// after pre-allocating slice for getting columns in UpdateCanonicalBricks
// BenchmarkAveragePlaythrough-12    	      14	  77828243 ns/op
// after allocating a buffer for obstacles only once and reusing it
// BenchmarkAveragePlaythrough-12    	      25	  46447304 ns/op
// after allocating a buffer for columns only once and reusing it
// BenchmarkAveragePlaythrough-12    	      28	  39575464 ns/op
// after pre-allocating the slice for holding Bricks
// BenchmarkAveragePlaythrough-12    	      28	  40986250 ns/op
// after using slices.SortFunc instead of sort.Slice (guided by memory profiler)
// BenchmarkAveragePlaythrough-12    	      30	  36079253 ns/op
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
