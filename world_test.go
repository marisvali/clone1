package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestWorld_RegressionTests(t *testing.T) {
	tests := GetFiles(os.DirFS(".").(FS), "regression-tests", "*.clone1")
	for _, test := range tests {
		playthrough := DeserializePlaythrough(ReadFile(test))
		expected := string(ReadFile(test + "-hash"))
		actual := RegressionId(playthrough)
		println(test)
		println(actual)
		println()
		assert.Equal(t, expected, actual)
	}
}

// TestWorld_ConvertRegressionTests should be used whenever we go from
// SimulationVersion = X to SimulationVersion = 999 or vice versa.
//
// Here is an example of the process I envision:
// A. I am at version 3 which I committed to
// B. I switch to 999 and I add features and play around
// C. I am done adding features and I start to harden my code.
// D. I add a regression test consisting of a complex playthrough.
// E. I refactor for clarity and maintainability.
// F. I refactor for performance.
// G. I add automated tests for geometric algorithms I am now sure I need.
// H. I add regression tests for various edge cases.
// I. I find bugs and missing things while testing edge cases. I add fixes,
// including changes to the simulation logic, which maybe invalidates some other
// regression tests, but maybe it doesn't.
// J. I commit to a final version 4 that is releasable.
//
// The steps above contain a point of friction: when do I change
// SimulationVersion in the code? I need to change the actual code and commit.
// Since all my regression tests are recorded playthroughs which contain the
// SimulationVersion, it means I should change SimulationVersion from 3 to 4
// as soon as I add my first regression test, at step D. Either this, or I have
// to redo all my regression tests when I reach step J and I commit to final
// version 4. But if I set SimulationVersion to 4 at step D, what do I do if I
// need to change some things quite drastically at step I? Then I have in my git
// history some commits that say SimulationVersion is 4, but really, are early
// attempts at 4, and later commits with SimulationVersion = 4 are really the
// ones that capture the logic of version 4.
// A simple way to cut through all of that is to make it easy to switch from
// 999 to 4 only at the very end, at step J. While I am still testing and
// tweaking, I am at 999. I only go to 4 when I am pretty sure I am done and I
// want to commit to 4. Of course, minor tweaks to 4 remain possible, and if
// major changes are necessary, I can just make a 5. But that should be very
// rare because I had a chance to play around and check everything while I was
// still on 999.
func TestWorld_ConvertRegressionTests(t *testing.T) {
	tests := GetFiles(os.DirFS(".").(FS), "regression-tests", "*.clone1")
	for _, test := range tests {
		playthrough := DeserializePlaythrough(ReadFile(test))
		fmt.Printf("%s changing SimulationVersion from %d to %d\n", test,
			playthrough.SimulationVersion, SimulationVersion)
		playthrough.SimulationVersion = SimulationVersion
		WriteFile(test, playthrough.Serialize())
	}
	assert.True(t, true)
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
// after pre-allocating the slice for holding BricksParams
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
// after assuming that rectangles are initialized properly and corner1 is always
// upper left and corner2 is always lower right
// BenchmarkAveragePlaythrough-12    	    1362	   8770731 ns/op
// after removing the optimization for filtering out which bricks are "near"
// in MarkFallingBricks
// BenchmarkAveragePlaythrough-12    	    1341	   8560720 ns/op
// after implementing a more rigorous filter for obstacles (with laptop plugged
// in, remember to unplug after battery recharges an rerun this measurement)
// BenchmarkAveragePlaythrough-12    	    1514	   7647922 ns/op
//
// Chained bricks were added and now the average playthrough has 21548 frames.
// The benchmark now, after adding chained bricks, before refactoring for
// performance:
// BenchmarkAveragePlaythrough-12    	     272	  43825387 ns/op
// After using a pre-allocated buffer in NoMoreMergesArePossible instead of
// allocating every time:
// BenchmarkAveragePlaythrough-12    	     434	  27863379 ns/op
// After disabling asserts for benchmark, as they are not relevant:
// BenchmarkAveragePlaythrough-12    	     447	  26703435 ns/op
func BenchmarkAveragePlaythrough(b *testing.B) {
	playthrough := DeserializePlaythrough(ReadFile("regression-tests/average-playthrough.clone1"))
	println(len(playthrough.History))
	for b.Loop() {
		world := NewWorldFromPlaythrough(playthrough)
		for i := range len(playthrough.History) {
			world.Step(playthrough.History[i])
		}
	}
}

func TestWorld_CreateNewRowOfBricks(t *testing.T) {
	RSeed(0)
	for range 10000 {
		var l Level
		l.BricksParams = append(l.BricksParams, BrickParams{
			Pos: CanonicalPosToPixelPos(Pt{5, 0}),
			Val: 29,
		})
		w := NewWorld(RInt(0, 10000), l)
		for {
			w.Step(PlayerInput{})
			if w.State == ComingUp {
				break
			}
		}
		for {
			w.Step(PlayerInput{})
			if w.State == Regular {
				break
			}
		}
		require.False(t, w.NoMoreMergesArePossible())
	}
}
