package main

import "testing"

// variant2:
// BenchmarkWorldSpeed-12    	   19231	     62310 ns/op
// variant3:
// BenchmarkWorldSpeed-12    	   15909	     74798 ns/op
// on variant 3 got 117000 ns/op -> 75202 ns/op for using GetLinePointsBuffered
func BenchmarkWorldSpeed(b *testing.B) {
	brickSize := Pt{100, 100}

	var brick Rectangle
	brick.Corner1 = Pt{100, 800}
	brick.Corner2 = brick.Corner1.Plus(brickSize)

	obstacles := make([]Rectangle, 30)
	for i := range obstacles {
		x := i % 5
		y := i / 5
		obstacles[i].Corner1 = Pt{x * 120, y * 120}
		obstacles[i].Corner2 = obstacles[i].Corner1.Plus(brickSize)
	}

	nMaxPixels := 100

	for b.Loop() {
		MoveRectTowardsTargetBlockedByRects(brick, Pt{0, 0}, nMaxPixels, obstacles)
	}
}
