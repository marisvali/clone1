package main

import (
	"github.com/stretchr/testify/assert"
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
		x := int64(i % 5)
		y := int64(i / 5)
		obstacles[i].Corner1 = Pt{x * 120, y * 120}
		obstacles[i].Corner2 = obstacles[i].Corner1.Plus(brickSize)
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
	r, nMaxPixels = MoveRect(r, Pt{targetPos.X, r.Corner1.Y}, nMaxPixels, obstacles)
	r, nMaxPixels = MoveRect(r, Pt{r.Corner1.X, targetPos.Y}, nMaxPixels, obstacles)
	return r
}

func TestRectContainsPt(t *testing.T) {
	r := Rectangle{Pt{10, 20}, Pt{30, 50}}
	assert.True(t, r.ContainsPt(Pt{10, 20}))
	assert.True(t, r.ContainsPt(Pt{10, 25}))
	assert.True(t, r.ContainsPt(Pt{15, 25}))
	assert.True(t, r.ContainsPt(Pt{30, 50}))
	assert.False(t, r.ContainsPt(Pt{9, 20}))
	assert.False(t, r.ContainsPt(Pt{10, 19}))
	assert.False(t, r.ContainsPt(Pt{31, 50}))
	assert.False(t, r.ContainsPt(Pt{30, 51}))
	assert.False(t, r.ContainsPt(Pt{31, 51}))
}

func TestRectIntersects(t *testing.T) {
	var r1, r2 Rectangle
	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(10, 20, 30, 50)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(10, 20, 11, 21)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(15, 25, 35, 55)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(5, 5, 15, 25)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(20, 20, 40, 50)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(10, 10, 30, 40)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(15, 10, 25, 60)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(15, 30, 25, 60)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(15, 25, 20, 30)
	assert.True(t, r1.Intersects(r2))
	assert.True(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(9, 19, 10, 20)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(30, 50, 60, 60)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(100, 200, 300, 500)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(0, 2, 3, 5)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(0, 0, 300, 10)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))

	r1 = NewRectangle(10, 20, 30, 50)
	r2 = NewRectangle(0, 0, 300, 20)
	assert.False(t, r1.Intersects(r2))
	assert.False(t, r2.Intersects(r1))
}
