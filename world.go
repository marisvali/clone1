package main

import (
	"math"
)

// World rules
// - Each brick belongs to a position in the matrix.
// - The position in the matrix to which the brick belongs to is determined like
// this: the matrix position for which the center is the closest to the center
// of a brick.
// - If a brick has no brick underneath it, it falls until it reaches a position
// in the matrix where there is a brick underneath it.
// - A brick has no brick underneath it if there is no brick at the matrix
// position underneath it for X seconds.
// - While bricks move, a brick cannot overlap another brick.
// - If a brick has no brick underneath it and it falls and that would mean it
// would overlap another brick which is currently being dragged, it pushes the
// dragged brick away.

type Brick struct {
	Val       int
	PosMat    Pt
	PosPixels Pt
}

type World struct {
	NCols           int
	NRows           int
	BrickPixelSize  int
	MarginPixelSize int
	Bricks          []Brick
	Dragging        *Brick
	DraggingOrigin  Pt
	DebugPts        []Pt
}

type PlayerInput struct {
	Pos          Pt
	JustPressed  bool
	JustReleased bool
}

func NewWorld() (w World) {
	w.NCols = 6
	w.NRows = 8
	w.MarginPixelSize = 30
	w.BrickPixelSize = (playWidth - (w.MarginPixelSize * (w.NCols + 1))) / w.NCols

	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			w.Bricks = append(w.Bricks, Brick{
				Val:       3,
				PosMat:    Pt{x, y},
				PosPixels: w.MatPosToPixelsPos(Pt{x, y}),
			})
		}
	}
	w.Dragging = nil
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelsPosToMatPos(pixelPos Pt) (matPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	matPos.X = int(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	matPos.Y = int(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) MatPosToPixelsPos(matPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = matPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (matPos.Y+1)*l
	return
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
		return // No line.
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

func (w *World) DraggingBrickHasValidPos() bool {
	if w.Dragging == nil {
		return false
	}
	r := w.BrickBounds(*w.Dragging)
	for j := range w.Bricks {
		if w.Dragging == &w.Bricks[j] {
			continue
		}
		r2 := w.BrickBounds(w.Bricks[j])
		if r.Intersects(r2) {
			return false
		}
	}
	return true
}

func (w *World) Step(input PlayerInput) {
	if input.JustPressed {
		// It should not be possible to be dragging anything already.
		if w.Dragging != nil {
			panic("wrong!")
		}

		// Check if there's any brick under the click.
		for i := range w.Bricks {
			p := w.Bricks[i].PosPixels
			brickSize := Pt{w.BrickPixelSize, w.BrickPixelSize}
			r := Rectangle{p, p.Plus(brickSize)}
			if r.ContainsPt(input.Pos) {
				w.Dragging = &w.Bricks[i]
				w.DraggingOrigin = input.Pos
				break
			}
		}
	}

	if input.JustReleased {
		if w.Dragging != nil {
			// Reset dragged brick's position.
			w.Dragging.PosMat = w.PixelsPosToMatPos(w.Dragging.PosPixels)
			w.Dragging.PosPixels = w.MatPosToPixelsPos(w.Dragging.PosMat)
		}
		w.Dragging = nil
	}

	if w.Dragging != nil {
		w.Dragging.PosPixels = w.ComputeDraggedBrickPosition(input)
	}
}

func RectIntersectsRects(r Rectangle, rects []Rectangle) bool {
	for _, r2 := range rects {
		if r.Intersects(r2) {
			return true
		}
	}
	return false
}

func (w *World) ComputeDraggedBrickPosition(input PlayerInput) Pt {
	offset := input.Pos.Minus(w.DraggingOrigin)
	targetPos := w.MatPosToPixelsPos(w.Dragging.PosMat).
		Plus(offset)
	oldPos := w.Dragging.PosPixels
	if oldPos == targetPos {
		// We are already at the target.
		return oldPos
	}

	// The overall logic of the movement is this:
	// - simulate the brick being dragged/moved towards the mouse position
	// - if the brick hits a wall or another brick, stop moving it on the
	// axis where it hit the obstacle, but continue moving on the other axis
	// as long as possible
	// - have some limited speed, don't make the brick just teleport to the
	// valid position in a single frame
	//
	// I can think of two ways to implement this logic:
	// 1. Find the right equations to solve in order to compute the target
	// position for the brick. (analytical solution)
	// 2. Move the brick in small steps and check if it collides with anything
	// after each move. (iterative solution)
	//
	// I will go for the iterative solution for now, because it's more
	// straightforward for me to come up with it. I'm simulating a process that
	// I imagine in an iterative way (the brick "moves towards the target").
	//
	// For the iterative solution, you can do it with floats or integers. I will
	// do it with integers and move the brick pixel by pixel. I do this because
	// I don't want to use floats in the world logic and I may be able to
	// afford the computational cost.
	//
	// Only travel a limited number of pixels, to have the effect of a brick's
	// "travel speed". The travel speed is not that noticeable when moving the
	// brick around in empty space. But if the brick was previously blocked by
	// something on its right and the user lifts it up to the point where now it
	// can go a long way through empty space to reach the mouse position, it
	// is very visible if the brick "travels" or "teleports" and the effect of
	// the brick travelling is more pleasant. It gives more of a feeling that it
	// is an actual solid object in solid space on which forces are acting.

	// First, get the set of rectangles the brick must not intersect.
	obstacles := make([]Rectangle, 0, len(w.Bricks))
	for j := range w.Bricks {
		if w.Dragging == &w.Bricks[j] {
			continue
		}
		obstacles = append(obstacles, w.BrickBounds(w.Bricks[j]))
	}

	bottom := playHeight - w.MarginPixelSize
	top := bottom - w.BrickPixelSize*w.NRows - w.MarginPixelSize*(w.NRows-1)
	left := w.MarginPixelSize
	right := playWidth - w.MarginPixelSize

	bottomRect := Rectangle{Pt{left, bottom}, Pt{right, bottom + 100}}
	topRect := Rectangle{Pt{left, top}, Pt{right, top - 100}}
	leftRect := Rectangle{Pt{left - 100, top}, Pt{left, bottom}}
	rightRect := Rectangle{Pt{right, top}, Pt{right + 100, bottom}}

	obstacles = append(obstacles, bottomRect)
	obstacles = append(obstacles, topRect)
	obstacles = append(obstacles, leftRect)
	obstacles = append(obstacles, rightRect)

	// First, go as far as possible towards the target, in a straight line.
	pts := GetLinePoints(Line{oldPos, targetPos})
	nMaxPixels := Min(len(pts), 70)
	brickSize := Pt{w.BrickPixelSize, w.BrickPixelSize}
	var i int
	for i = 1; i < nMaxPixels; i++ {
		brick := Rectangle{pts[i], pts[i].Plus(brickSize)}
		if RectIntersectsRects(brick, obstacles) {
			break
		}
	}

	// At this point, pts[i-1] is the last valid position either because
	// we reached the target, or we travelled the maximum number of pixels
	// or we hit an obstacle at pt[i].
	lastValidPos := pts[i-1]

	// Did we reach the target or travel the maximum number of pixels?
	if i == nMaxPixels {
		// Yes, return the last valid pos.
		return lastValidPos
	}
	// No, which means we hit an obstacle. Which means we should still try to
	// travel the rest of the pixels we have left either on X or on Y.
	nPixelsLeft := nMaxPixels - i

	// Move towards the target X until we reach the target X, or finish the
	// pixels we have left or we hit another obstacle.
	if lastValidPos.X != targetPos.X {
		pos := lastValidPos
		incX := 1
		if pos.X > targetPos.X {
			incX = -1
		}
		for {
			pos.X += incX
			brick := Rectangle{pos, pos.Plus(brickSize)}
			if RectIntersectsRects(brick, obstacles) {
				// If we hit an obstacle after the first movement, we should try to
				// move on Y.
				if lastValidPos == pts[i-1] {
					break
				} else {
					return lastValidPos
				}
			}
			// If we got here, we had at least one valid movement on X, which
			// means further movement on Y is out of the question.
			lastValidPos = pos
			nPixelsLeft--
			if nPixelsLeft == 0 {
				return lastValidPos
			}
			if pos.X == targetPos.X {
				return lastValidPos
			}
		}
	}

	// Move towards the target Y until we reach the target Y, or finish the
	// pixels we have left or we hit another obstacle.
	if lastValidPos.Y != targetPos.Y {
		pos := lastValidPos
		incY := 1
		if pos.Y > targetPos.Y {
			incY = -1
		}
		for {
			pos.Y += incY
			brick := Rectangle{pos, pos.Plus(brickSize)}
			if RectIntersectsRects(brick, obstacles) {
				return lastValidPos
			}
			// If we got here, we had at least one valid movement on X, which
			// means further movement on Y is out of the question.
			lastValidPos = pos
			nPixelsLeft--
			if nPixelsLeft == 0 {
				return lastValidPos
			}
			if pos.Y == targetPos.Y {
				return lastValidPos
			}
		}
	}
	return lastValidPos
}

func (w *World) BrickBounds(b Brick) (r Rectangle) {
	r.Corner1 = b.PosPixels
	r.Corner2 = b.PosPixels
	r.Corner2.X += w.BrickPixelSize
	r.Corner2.Y += w.BrickPixelSize
	return
}
