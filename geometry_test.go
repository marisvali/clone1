package main

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

// BenchmarkMoveRect-12    	   96234	     12439 ns/op
func BenchmarkMoveRect(b *testing.B) {
	brickSize := Pt{100, 100}

	pt := Pt{10000, 800}
	brick := NewRectangle(pt, pt.Plus(brickSize))

	obstacles := make([]Rectangle, 30)
	for i := range obstacles {
		x := int64(i % 5)
		y := int64(i / 5)
		pt = Pt{x * 120, y * 120}
		obstacles[i] = NewRectangle(pt, pt.Plus(brickSize))
	}

	nMaxPixels := int64(100)

	for b.Loop() {
		// If I put the code here directly instead of wrapped into a function,
		// it gets optimized away for some reason.
		f(brick, Pt{0, 0}, nMaxPixels, obstacles)
	}
}

func f(r Rectangle, targetPos Pt, nMaxPixels int64, obstacles []Rectangle) Rectangle {
	r, nMaxPixels = MoveRect(r, targetPos, nMaxPixels, obstacles)
	r, nMaxPixels = MoveRect(r, Pt{targetPos.X, r.Min.Y}, nMaxPixels, obstacles)
	r, nMaxPixels = MoveRect(r, Pt{r.Min.X, targetPos.Y}, nMaxPixels, obstacles)
	return r
}

func TestRectContainsPt(t *testing.T) {
	r := NewRectangle(Pt{10, 20}, Pt{30, 50})
	assert.True(t, r.ContainsPt(Pt{10, 20}))
	assert.True(t, r.ContainsPt(Pt{10, 25}))
	assert.True(t, r.ContainsPt(Pt{15, 25}))
	assert.False(t, r.ContainsPt(Pt{30, 50}))
	assert.False(t, r.ContainsPt(Pt{9, 20}))
	assert.False(t, r.ContainsPt(Pt{10, 19}))
	assert.False(t, r.ContainsPt(Pt{31, 50}))
	assert.False(t, r.ContainsPt(Pt{30, 51}))
	assert.False(t, r.ContainsPt(Pt{31, 51}))
}

func TestRectIntersects(t *testing.T) {
	var r1, r2 Rectangle
	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{10, 20}, Pt{11, 21})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{15, 25}, Pt{35, 55})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{5, 5}, Pt{15, 25})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{20, 20}, Pt{40, 50})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{10, 10}, Pt{30, 40})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{15, 10}, Pt{25, 60})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{15, 30}, Pt{25, 60})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{15, 25}, Pt{20, 30})
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{9, 19}, Pt{10, 20})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{30, 50}, Pt{60, 60})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{100, 200}, Pt{300, 500})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{0, 2}, Pt{3, 5})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{0, 0}, Pt{300, 10})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(Pt{10, 20}, Pt{30, 50})
	r2 = NewRectangle(Pt{0, 0}, Pt{300, 20})
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))
}

func TestGetLinePointsAll(t *testing.T) {
	var start, end Pt
	var nMaxPts int64
	var actualPts, expectedPts []Pt

	start, end, nMaxPts = Pt{0, 0}, Pt{10, 10}, 3
	expectedPts = []Pt{{0, 0}, {1, 1}, {2, 2}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{10, 10}, Pt{0, 0}, 3
	expectedPts = []Pt{{10, 10}, {9, 9}, {8, 8}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{0, 0}, Pt{10, 0}, 3
	expectedPts = []Pt{{0, 0}, {1, 0}, {2, 0}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{10, 0}, Pt{0, 0}, 3
	expectedPts = []Pt{{10, 0}, {9, 0}, {8, 0}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{0, 0}, Pt{0, 10}, 3
	expectedPts = []Pt{{0, 0}, {0, 1}, {0, 2}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{0, 10}, Pt{0, 0}, 3
	expectedPts = []Pt{{0, 10}, {0, 9}, {0, 8}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	start, end, nMaxPts = Pt{0, 0}, Pt{10, 4}, 5
	expectedPts = []Pt{{0, 0}, {1, 0}, {2, 0}, {3, 1}, {4, 1}}
	actualPts = GetLinePoints(start, end, nMaxPts)
	assert.Equal(t, expectedPts, actualPts)
	OneTestGetLinePoints(t, start, end, nMaxPts)

	OneTestGetLinePoints(t, Pt{10, 30}, Pt{1000, 0}, 800)
	OneTestGetLinePoints(t, Pt{1000, 0}, Pt{10, 30}, 800)
	OneTestGetLinePoints(t, Pt{0, 1000}, Pt{-100, 324}, 800)
	OneTestGetLinePoints(t, Pt{-993, 193}, Pt{3922, 4}, 800)
}

func OneTestGetLinePoints(t *testing.T, start, end Pt, nMaxPts int64) {
	pts := GetLinePoints(start, end, nMaxPts)

	// Test that nMaxPts is respected.
	assert.LessOrEqual(t, len(pts), int(nMaxPts))

	// Test that the start fits.
	assert.Equal(t, start, pts[0])

	// Test that the progression from point to point is correct.
	for i := 1; i < len(pts); i++ {
		// The next point is different from the previous one.
		assert.NotEqual(t, pts[i], pts[i-1])

		// The next point is touching the previous one.
		stepDist := pts[i].SquaredDistTo(pts[i-1])
		assert.True(t, stepDist == 1 || stepDist == 2)

		// The next point is closer to the end than the previous one.
		prevDistToEnd := pts[i-1].SquaredDistTo(end)
		distToEnd := pts[i].SquaredDistTo(end)
		assert.Less(t, distToEnd, prevDistToEnd)

		// Compute the proper point in floats.
		// Compute the number of points between start and end.
		diff := end.Minus(start)
		numPts := Max(Abs(diff.X), Abs(diff.Y))
		// How far along the line are we, at point i?
		factor := float64(i) / float64(numPts)
		// What is the point at this distance in the line?
		properX := float64(start.X) + float64(diff.X)*factor
		properY := float64(start.Y) + float64(diff.Y)*factor
		// Make sure the pixelated point is as close to the proper point as
		// possible.
		assert.Less(t, math.Abs(float64(pts[i].X)-properX), 1.0)
		assert.Less(t, math.Abs(float64(pts[i].Y)-properY), 1.0)
	}
}

func TestMoveRect(t *testing.T) {
	obstacles := []Rectangle{}
	obstacles = append(obstacles, NewRectangle(Pt{0, 0}, Pt{10, 10}))
	obstacles = append(obstacles, NewRectangle(Pt{20, 0}, Pt{30, 10}))
	obstacles = append(obstacles, NewRectangle(Pt{0, 20}, Pt{0, 30}))
	obstacles = append(obstacles, NewRectangle(Pt{20, 20}, Pt{30, 30}))

	var pos, size, targetPos, newPos Pt
	var r, newR Rectangle
	var nMaxPixels, nExpectedPixelsLeft, nPixelsLeft int64

	// Going left to right.
	pos = Pt{-20, 0}
	size = Pt{10, 10}
	targetPos = Pt{100, 0}
	nMaxPixels = 100
	newPos = Pt{-10, 0}
	nExpectedPixelsLeft = 90
	r = NewRectangle(pos, pos.Plus(size))
	newR, nPixelsLeft = MoveRect(r, targetPos, nMaxPixels, obstacles)
	assert.Equal(t, newR, NewRectangle(newPos, newPos.Plus(size)))
	assert.Equal(t, nExpectedPixelsLeft, nPixelsLeft)
	OneTestMoveRect(t, r, targetPos, nMaxPixels, obstacles)

	// Going right to left.
	pos = Pt{100, 0}
	size = Pt{10, 10}
	targetPos = Pt{0, 0}
	nMaxPixels = 100
	newPos = Pt{30, 0}
	nExpectedPixelsLeft = 30
	r = NewRectangle(pos, pos.Plus(size))
	newR, nPixelsLeft = MoveRect(r, targetPos, nMaxPixels, obstacles)
	assert.Equal(t, newR, NewRectangle(newPos, newPos.Plus(size)))
	assert.Equal(t, nExpectedPixelsLeft, nPixelsLeft)
	OneTestMoveRect(t, r, targetPos, nMaxPixels, obstacles)

	// Going bottom to top.
	pos = Pt{-5, -50}
	size = Pt{10, 10}
	targetPos = Pt{-5, 100}
	nMaxPixels = 100
	newPos = Pt{-5, -10}
	nExpectedPixelsLeft = 60
	r = NewRectangle(pos, pos.Plus(size))
	newR, nPixelsLeft = MoveRect(r, targetPos, nMaxPixels, obstacles)
	assert.Equal(t, newR, NewRectangle(newPos, newPos.Plus(size)))
	assert.Equal(t, nExpectedPixelsLeft, nPixelsLeft)
	OneTestMoveRect(t, r, targetPos, nMaxPixels, obstacles)

	// Going top to bottom.
	pos = Pt{-5, 100}
	size = Pt{10, 10}
	targetPos = Pt{-5, 0}
	nMaxPixels = 100
	newPos = Pt{-5, 30}
	nExpectedPixelsLeft = 30
	r = NewRectangle(pos, pos.Plus(size))
	newR, nPixelsLeft = MoveRect(r, targetPos, nMaxPixels, obstacles)
	assert.Equal(t, newR, NewRectangle(newPos, newPos.Plus(size)))
	assert.Equal(t, nExpectedPixelsLeft, nPixelsLeft)
	OneTestMoveRect(t, r, targetPos, nMaxPixels, obstacles)

	// Diagonals.
	pos = Pt{-500, 100}
	size = Pt{5, 5}
	targetPos = Pt{-5, 0}
	nMaxPixels = 1000
	OneTestMoveRect(t, NewRectangle(pos, pos.Plus(size)), targetPos, nMaxPixels,
		obstacles)

	pos = Pt{300, 500}
	size = Pt{5, 5}
	targetPos = Pt{-5, 0}
	nMaxPixels = 1000
	OneTestMoveRect(t, NewRectangle(pos, pos.Plus(size)), targetPos, nMaxPixels,
		obstacles)

	pos = Pt{-300, -250}
	size = Pt{5, 5}
	targetPos = Pt{-5, 0}
	nMaxPixels = 1000
	OneTestMoveRect(t, NewRectangle(pos, pos.Plus(size)), targetPos, nMaxPixels,
		obstacles)

	pos = Pt{300, -250}
	size = Pt{5, 5}
	targetPos = Pt{-5, 0}
	nMaxPixels = 1000
	OneTestMoveRect(t, NewRectangle(pos, pos.Plus(size)), targetPos, nMaxPixels,
		obstacles)
}

func OneTestMoveRect(t *testing.T, r Rectangle, targetPos Pt, nMaxPixels int64,
	obstacles []Rectangle) {
	newR, nPixelsLeft := MoveRect(r, targetPos, nMaxPixels, obstacles)

	// Check that the pixels left is correct.
	dif := newR.Min.Minus(r.Min)
	pixelsTravelled := Max(Abs(dif.X), Abs(dif.Y))
	assert.Equal(t, nMaxPixels-pixelsTravelled, nPixelsLeft)

	// Check that the new rectangle doesn't intersect obstacles.
	assert.False(t, RectIntersectsRects(newR, obstacles))

	// Check that by travelling just a little bit more towards the target, the
	// new rectangle will intersect something.
	var dir Pt
	if dif.X != 0 {
		dir.X = dif.X / Abs(dif.X)
	}
	if dif.Y != 0 {
		dir.Y = dif.Y / Abs(dif.Y)
	}

	pos2 := newR.Min.Plus(dir)
	newR2 := Rectangle{pos2, pos2.Plus(newR.Size())}
	assert.True(t, RectIntersectsRects(newR2, obstacles))
}
