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
// after refactoring the code for marking falling bricks
// BenchmarkAveragePlaythrough-12    	      44	  25425711 ns/op
// after precomputing derived values and using precomputations in UpdateCanonicalBricks
// BenchmarkAveragePlaythrough-12    	      49	  23793808 ns/op
// after computing derived values only when bricks are moved and also use integer math for PixelPosToCanonicalPos instead of float math
// BenchmarkAveragePlaythrough-12    	      49	  25785637 ns/op
// after making benchmark last 10s instead of 1s to get more accurate results:
// BenchmarkAveragePlaythrough-12    	     500	  23831522 ns/op
// after MoveBrick returns immediately if the position is the same, to avoid
// unnecessary calls to UpdateDerivedValues in UpdateCanonicalBricks (guided by CPU profiler)
// BenchmarkAveragePlaythrough-12    	     553	  21457118 ns/op
// after GetObstacles only returns bricks that are close
// BenchmarkAveragePlaythrough-12    	     939	  12716588 ns/op
// after reverting to simpler implementation for PixelPosToCanonicalPos
// BenchmarkAveragePlaythrough-12    	     915	  13114277 ns/op
// after optimizing FindMergingBricks
// BenchmarkAveragePlaythrough-12    	    1141	  10529063 ns/op
// after reverting the change to precompute derived values
// BenchmarkAveragePlaythrough-12    	     648	  18501062 ns/op
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
