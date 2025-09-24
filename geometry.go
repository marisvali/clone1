package main

type Line struct {
	Start Pt
	End   Pt
}

type Circle struct {
	Center   Pt
	Diameter int
}

type Square struct {
	Center Pt
	Size   int
}

type Rectangle struct {
	Corner1 Pt
	Corner2 Pt
}

func Abs(x int) int {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func Min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func Max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func MinMax(x int, y int) (int, int) {
	if x < y {
		return x, y
	} else {
		return y, x
	}
}

func (r *Rectangle) Width() int {
	return Abs(r.Corner1.X - r.Corner2.X)
}

func (r *Rectangle) Height() int {
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

// GetLinePoints computes a list of points that lie between the start and end
// of a line. The points all have integer coordinates and they are continuous
// (pixel k touches pixel k-1). This algorithm is useful if you want to draw a
// line on a bitmap, for example. Mathematically speaking, there is an infinite
// number of points on a line, and their coordinates are almost always not
// integers. So we need to decide which pixels best approximate the actual line.
// Important: the points are ordered and go from line start to line end.
func GetLinePoints(l Line) (pts []Pt) {
	x1 := l.Start.X
	y1 := l.Start.Y
	x2 := l.End.X
	y2 := l.End.Y

	dx := x2 - x1
	dy := y2 - y1
	if dx == 0 && dy == 0 {
		pts = append(pts, l.Start)
		return
	}

	if Abs(dx) > Abs(dy) {
		inc := dx / Abs(dx)
		for x := x1; x != x2; x += inc {
			y := y1 + (x-x1)*dy/dx
			pts = append(pts, Pt{x, y})
		}
	} else {
		inc := dy / Abs(dy)
		for y := y1; y != y2; y += inc {
			x := x1 + (y-y1)*dx/dy
			pts = append(pts, Pt{x, y})
		}
	}
	return
}

var buffer []Pt = make([]Pt, 100000)

func GetLinePointsBuffered(l Line) []Pt {
	n := 0

	x1 := l.Start.X
	y1 := l.Start.Y
	x2 := l.End.X
	y2 := l.End.Y

	dx := x2 - x1
	dy := y2 - y1
	if dx == 0 && dy == 0 {
		buffer[n] = l.Start
		n++
		return buffer[:n]
	}

	if Abs(dx) > Abs(dy) {
		inc := dx / Abs(dx)
		for x := x1; x != x2; x += inc {
			y := y1 + (x-x1)*dy/dx
			buffer[n] = Pt{x, y}
			n++
		}
	} else {
		inc := dy / Abs(dy)
		for y := y1; y != y2; y += inc {
			x := x1 + (y-y1)*dx/dy
			buffer[n] = Pt{x, y}
			n++
		}
	}
	return buffer[:n]
}

func MoveRectUntilBlockedByRects(r Rectangle, targetPos Pt,
	nMaxPixels int, obstacles []Rectangle) (newPos Pt) {

	rSize := Pt{r.Width(), r.Height()}
	lastValidPos := r.Corner1
	var i int

	// First, go as far as possible towards the target, in a straight line.
	pts := GetLinePointsBuffered(Line{lastValidPos, targetPos})
	nMaxPixels = Min(len(pts), nMaxPixels)
	for i = 1; i < nMaxPixels; i++ {
		newR := Rectangle{pts[i], pts[i].Plus(rSize)}
		if RectIntersectsRects(newR, obstacles) {
			break
		}
	}

	// At this point, pts[i-1] is the last valid position either because
	// we reached the target, or we travelled the maximum number of pixels
	// or we hit an obstacle at pt[i].
	return pts[i-1]
}

func MoveRectTowardsTargetBlockedByRects(r Rectangle, targetPos Pt,
	nMaxPixels int, obstacles []Rectangle) (newPos Pt) {

	rSize := Pt{r.Width(), r.Height()}
	lastValidPos := r.Corner1
	var i int

	// First, go as far as possible towards the target, in a straight line.
	pts := GetLinePointsBuffered(Line{lastValidPos, targetPos})
	nMaxPixels = Min(len(pts), nMaxPixels)
	for i = 1; i < nMaxPixels; i++ {
		newR := Rectangle{pts[i], pts[i].Plus(rSize)}
		if RectIntersectsRects(newR, obstacles) {
			break
		}
	}

	// At this point, pts[i-1] is the last valid position either because
	// we reached the target, or we travelled the maximum number of pixels
	// or we hit an obstacle at pt[i].
	lastValidPos = pts[i-1]

	// Now, go towards the target's X as much as possible.
	pts = GetLinePointsBuffered(Line{lastValidPos, Pt{targetPos.X, lastValidPos.Y}})

	nMaxPixels = Min(len(pts), nMaxPixels)
	for i = 1; i < nMaxPixels; i++ {
		newR := Rectangle{pts[i], pts[i].Plus(rSize)}
		if RectIntersectsRects(newR, obstacles) {
			break
		}
	}
	lastValidPos = pts[i-1]

	// Now, go towards the target's Y as much as possible.
	pts = GetLinePointsBuffered(Line{lastValidPos, Pt{lastValidPos.X, targetPos.Y}})
	nMaxPixels = Min(len(pts), nMaxPixels)
	for i = 1; i < nMaxPixels; i++ {
		newR := Rectangle{pts[i], pts[i].Plus(rSize)}
		if RectIntersectsRects(newR, obstacles) {
			break
		}
	}
	lastValidPos = pts[i-1]
	return lastValidPos
}

func MoveRectTowardsTargetBlockedByRects2(r Rectangle, targetPos Pt,
	nMaxPixels int, obstacles []Rectangle) (newPos Pt) {

	brickSize := Pt{r.Width(), r.Height()}
	lastValidPos := r.Corner1
	var i int

	// First, go as far as possible towards the target, in a straight line.
	pts := GetLinePoints(Line{lastValidPos, targetPos})
	nMaxPixels = Min(len(pts), nMaxPixels)
	for i = 1; i < nMaxPixels; i++ {
		newR := Rectangle{pts[i], pts[i].Plus(brickSize)}
		if RectIntersectsRects(newR, obstacles) {
			break
		}
	}

	// At this point, pts[i-1] is the last valid position either because
	// we reached the target, or we travelled the maximum number of pixels
	// or we hit an obstacle at pt[i].
	lastValidPos = pts[i-1]
	nPixelsLeft := nMaxPixels - i

	// Try to move on both X and Y until the target X or Y is reached, or we
	// have travelled all the pixels we have left, or we can no longer move on
	// X or Y. We expect to only be able to move on X or Y, not both at once,
	// but we don't know which one at the start.
	incX := 1
	if lastValidPos.X > targetPos.X {
		incX = -1
	}
	incY := 1
	if lastValidPos.Y > targetPos.Y {
		incY = -1
	}
	canMoveOnX := true
	canMoveOnY := true
	for {
		if canMoveOnX {
			if lastValidPos.X == targetPos.X {
				canMoveOnX = false
			} else {
				pos := lastValidPos.Plus(Pt{incX, 0})
				brick := Rectangle{pos, pos.Plus(brickSize)}
				if RectIntersectsRects(brick, obstacles) {
					canMoveOnX = false
				} else {
					lastValidPos = pos
				}
			}
		}
		if canMoveOnY {
			if lastValidPos.Y == targetPos.Y {
				canMoveOnY = false
			} else {
				pos := lastValidPos.Plus(Pt{0, incY})
				brick := Rectangle{pos, pos.Plus(brickSize)}
				if RectIntersectsRects(brick, obstacles) {
					canMoveOnY = false
				} else {
					lastValidPos = pos
				}
			}
		}
		nPixelsLeft--
		if nPixelsLeft == 0 {
			return lastValidPos
		}
		if !canMoveOnX && !canMoveOnY {
			return lastValidPos
		}
	}
}
