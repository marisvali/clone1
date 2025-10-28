package main

import (
	"github.com/stretchr/testify/assert"
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
