package main

import (
	"testing"
)

// BenchmarkMoveRect-12    	   96234	     12439 ns/op
func BenchmarkMoveRect(b *testing.B) {
	brickSize := Pt{100, 100}

	var brick Rectangle
	brick.Corner1 = Pt{10000, 800}
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
		// If I put the code here directly instead of wrapped into a function,
		// it gets optimized away for some reason.
		f(brick, Pt{0, 0}, nMaxPixels, obstacles)
	}
}

func f(r Rectangle, targetPos Pt, nMaxPixels int, obstacles []Rectangle) Rectangle {
	r, nMaxPixels = MoveRect(r, targetPos, nMaxPixels, obstacles)
	r, nMaxPixels = MoveRect(r, Pt{targetPos.X, r.Corner1.Y}, nMaxPixels, obstacles)
	r, nMaxPixels = MoveRect(r, Pt{r.Corner1.X, targetPos.Y}, nMaxPixels, obstacles)
	return r
}
