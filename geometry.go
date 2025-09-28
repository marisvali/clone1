package main

import "fmt"

type Line struct {
	Start Pt
	End   Pt
}

type Circle struct {
	Center   Pt
	Diameter int64
}

type Square struct {
	Center Pt
	Size   int64
}

type Rectangle struct {
	Corner1 Pt
	Corner2 Pt
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func Min(x int64, y int64) int64 {
	if x < y {
		return x
	} else {
		return y
	}
}

func Max(x int64, y int64) int64 {
	if x > y {
		return x
	} else {
		return y
	}
}

func MinMax(x int64, y int64) (int64, int64) {
	if x < y {
		return x, y
	} else {
		return y, x
	}
}

func (r *Rectangle) Width() int64 {
	return Abs(r.Corner1.X - r.Corner2.X)
}

func (r *Rectangle) Height() int64 {
	return Abs(r.Corner1.Y - r.Corner2.Y)
}

func (r *Rectangle) Min() Pt {
	return Pt{Min(r.Corner1.X, r.Corner2.X), Min(r.Corner1.Y, r.Corner2.Y)}
}

func (r *Rectangle) Max() Pt {
	return Pt{Max(r.Corner1.X, r.Corner2.X), Max(r.Corner1.Y, r.Corner2.Y)}
}

func (r *Rectangle) ContainsPt(pt Pt) bool {
	minX, maxX := MinMax(r.Corner1.X, r.Corner2.X)
	minY, maxY := MinMax(r.Corner1.Y, r.Corner2.Y)
	return pt.X >= minX && pt.X <= maxX && pt.Y >= minY && pt.Y <= maxY
}

func (r *Rectangle) Intersects(other Rectangle) bool {
	minX1, maxX1 := MinMax(r.Corner1.X, r.Corner2.X)
	minY1, maxY1 := MinMax(r.Corner1.Y, r.Corner2.Y)
	minX2, maxX2 := MinMax(other.Corner1.X, other.Corner2.X)
	minY2, maxY2 := MinMax(other.Corner1.Y, other.Corner2.Y)
	return minX1 < maxX2 && maxX1 > minX2 && minY1 < maxY2 && maxY1 > minY2
}

// linePointsBufferSize is an arbitrary limit for GetLinePoints. Change its
// value to accommodate your needs. The only concern is to have something that
// doesn't eat up RAM unnecessarily but is good enough for everything the game
// needs.
const linePointsBufferSize = 10000

// linePointsBuffer is a buffer allocated only once and reused by GetLinePoints.
var linePointsBuffer = make([]Pt, linePointsBufferSize)

// GetLinePoints computes a list of points that lie between the start and end
// of a line. The points all have integer coordinates and they are continuous
// (pixel k touches pixel k-1). This algorithm is useful if you want to draw a
// line on a bitmap, for example. Mathematically speaking, there is an infinite
// number of points on a line, and their coordinates are almost always not
// integers. So we need to decide which pixels best approximate the actual line.
// GetLinePoints does the standard approximation that you might see in something
// like Windows Paint.
// Important: the points are ordered and go from line start to line end.
func GetLinePoints(start Pt, end Pt, nMaxPts int64) []Pt {
	if nMaxPts > linePointsBufferSize {
		panic(fmt.Errorf("got nMaxPts = %d but can only handle at most %d "+
			"points", nMaxPts, linePointsBufferSize))
	}

	n := int64(0)
	x1 := start.X
	y1 := start.Y
	x2 := end.X
	y2 := end.Y

	dx := x2 - x1
	dy := y2 - y1
	// Check if dx or dy are zero, to avoid division by zero further down the
	// line.
	if dx == 0 && dy == 0 {
		// If start and end are the same, return a single point.
		linePointsBuffer[n] = start
		n++
		return linePointsBuffer[:n]
	}

	if Abs(dx) > Abs(dy) {
		// The line is longer on X than on Y. Then we need exactly one pixel for
		// each X coordinate. What's left is to compute the corresponding Y for
		// each X.
		inc := dx / Abs(dx) // I use inc, which might be +1 or -1, because it is
		// important for me to go from start to end, not just from min to max.
		x2 += inc // We want the end point to be added to the line if it is
		// within nMaxPts. The condition for x must be x != x2 because we don't
		// know if inc is 1 or -1 so we cannot do x <= x2 or x >= x2. So, just
		// increase x2 by inc.
		for x := x1; x != x2 && n < nMaxPts; x += inc {
			// I intentionally don't compute dy/dx once and reuse it because
			// that would mean doing floating point operations. I want to do
			// only integer operations.
			y := y1 + (x-x1)*dy/dx
			linePointsBuffer[n] = Pt{x, y}
			n++
		}
	} else {
		// The comments for X apply here as well, with X and Y interchanged.
		inc := dy / Abs(dy)
		y2 += inc
		for y := y1; y != y2 && n < nMaxPts; y += inc {
			x := x1 + (y-y1)*dx/dy
			linePointsBuffer[n] = Pt{x, y}
			n++
		}
	}
	return linePointsBuffer[:n]
}

// RectIntersectsRects is a utility function that checks if a rectangle
// intersects any of a list of rectangles.
func RectIntersectsRects(r Rectangle, rects []Rectangle) bool {
	for _, r2 := range rects {
		if r.Intersects(r2) {
			return true
		}
	}
	return false
}

// MoveRect computes a rectangle newR the size of r as if r was moved in a
// straight line towards targetPos until:
// - it reached targetPos or
// - it moved for nMaxPixels or
// - it intersected an obstacle
// The position of the rectangle is r.Corner1. If r can reach the targetPos,
// then newR.Corner1 == targetPos.
func MoveRect(r Rectangle, targetPos Pt, nMaxPixels int64,
	obstacles []Rectangle) (newR Rectangle, nPixelsLeft int64) {

	// Compute the pixels along the line from the start position to the target
	// position. We do nMaxPixels+1 because the first pixel in the line is the
	// current position, which we do not consider a movement.
	pts := GetLinePoints(r.Corner1, targetPos, nMaxPixels+1)

	// Move the rectangle pixel by pixel and check if it collides with any of
	// the obstacles.
	rSize := Pt{r.Width(), r.Height()}
	var i int64
	for i = 1; i < int64(len(pts)); i++ {
		r = Rectangle{pts[i], pts[i].Plus(rSize)}
		if RectIntersectsRects(r, obstacles) {
			break
		}
	}

	// At this point, pts[i-1] is the last valid position either because
	// we reached the target, or we travelled the maximum number of pixels
	// or we hit an obstacle at pt[i].
	return Rectangle{pts[i-1], pts[i-1].Plus(rSize)}, nMaxPixels - i + 1
}
