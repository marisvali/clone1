package main

import "testing"

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
