package main

import (
	"fmt"
	"math"
)

// World rules (physics)
// ---------------------
//
// Terminology:
// - Slot: the space of the world is a matrix of 6x8 rectangles separated by some
// margins. These rectangular places are called slots. Some are empty, some will
// have bricks in them.
// - Brick: rectangular objects that fit inside slots that the player can drag
// around and match.
// - Canonical position: a position in the matrix of slots. For example
// (2, 0) means the third brick at the bottom row.
// - Pixel position: the position of a brick in terms of pixels, not slots. The
// pixel position can place a brick between slots.
// - Canonical pixel position: a position in the matrix of slots converted to
// pixel coordinates. For example depending on brick size and margin sizes the
// canonical position (2, 0) can be turned into the canonical pixel position
// (230, 30).
// - Static brick: a brick that is currently not moving on its own and not being
// dragged around by the player.
// - Dragging brick: a brick that the player has clicked on and is currently
// dragging around the world space.
// - Falling brick: a brick that is not dragged by the player and is currently
// falling because it has nothing underneath it.
//
// Rules for dragging a brick:
// - When the player clicks on a brick, it becomes a dragging brick.
// - When the player releases the click on a brick, the dragging brick moves to
// its nearest canonical position and becomes a static or falling brick.
// - When the player moves the mouse while dragging a brick, the brick moves
// towards the mouse position.
// - If the dragged brick hits a wall or another brick, it stops moving on the
// axis where it hit the obstacle, but it continues moving on the other axis
// as long as possible.
// - The dragged brick goes towards the mouse cursor with some limited speed,
// it doesn't just teleport to the nearest valid position in a single frame.
//
// Rules for falling:
// - A static brick becomes a falling brick when the slot underneath it is
// completely free (does not contain any part of any other brick, static or
// dragging).
// - A falling brick stops being a falling brick when it intersects another
// brick.
// - When a falling brick stops being a falling brick, it moves to its canonical
// position automatically.
// - Bricks fall with acceleration.

type Brick struct {
	Val          int
	PixelPos     Pt
	Falling      bool
	FallingSpeed int
}

type World struct {
	NCols                 int
	NRows                 int
	BrickPixelSize        int
	MarginPixelSize       int
	BrickFallAcceleration int
	Bricks                []Brick
	Dragging              *Brick
	DraggingOffset        Pt
	DebugPts              []Pt
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
	w.BrickFallAcceleration = 2

	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			w.Bricks = append(w.Bricks, Brick{
				Val:      (x+y)%3 + 1,
				PixelPos: w.CanonicalPosToPixelsPos(Pt{x, y}),
			})
		}
	}
	// for y := 0; y < 1; y++ {
	// 	for x := 0; x < 3; x++ {
	// 		w.Bricks = append(w.Bricks, Brick{
	// 			Val: 3,
	// 			// PosMat:    Pt{x, y},
	// 			PixelPos: w.CanonicalPosToPixelsPos(Pt{x, y}),
	// 		})
	// 	}
	// }
	w.Dragging = nil
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelsPosToCanonicalPos(pixelPos Pt) (matPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	matPos.X = int(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	matPos.Y = int(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) CanonicalPosToPixelsPos(matPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = matPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (matPos.Y+1)*l
	return
}

func (w *World) Step(input PlayerInput) {
	if input.JustPressed {
		// It should not be possible to be dragging anything already.
		if w.Dragging != nil {
			panic("wrong!")
		}

		// Check if there's any brick under the click.
		for i := range w.Bricks {
			p := w.Bricks[i].PixelPos
			brickSize := Pt{w.BrickPixelSize, w.BrickPixelSize}
			r := Rectangle{p, p.Plus(brickSize)}
			if r.ContainsPt(input.Pos) {
				w.Dragging = &w.Bricks[i]
				w.DraggingOffset = p.Minus(input.Pos)
				break
			}
		}
	}

	if input.JustReleased {
		if w.Dragging != nil {
			// Reset dragged brick's position.
			// w.Dragging.PosMat = w.PixelsPosToCanonicalPos(w.Dragging.PixelPos)
			w.Dragging.PixelPos = w.CanonicalPosToPixelsPos(w.PixelsPosToCanonicalPos(w.Dragging.PixelPos))
		}
		w.Dragging = nil
	}

	if w.Dragging != nil {
		w.Dragging.PixelPos = w.ComputeDraggedBrickPosition(input)
	}

	w.SetBricksToFalling()
	w.AdvanceFallingBricks()

	// check if we got in a bad state
	{
		for i := range w.Bricks {
			obstacles := w.GetObstacles(&w.Bricks[i])
			brick := w.BrickBounds(w.Bricks[i].PixelPos)
			if RectIntersectsRects(brick, obstacles) {
				panic("wrong!")
			}
		}
	}
}

func (w *World) SetBricksToFalling() {
	// Mark falling bricks.
	for i := range w.Bricks {
		// Skip the dragging brick.
		b := &w.Bricks[i]
		if b == w.Dragging {
			continue
		}

		// Skip already falling bricks.
		if b.Falling {
			continue
		}

		// Assert that if a brick is not falling, it is at its canonical pos.
		{
			cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
			pPos := w.CanonicalPosToPixelsPos(cPos)
			if pPos != b.PixelPos {
				panic(fmt.Errorf("brick is not at its canonical pos"))
			}
		}

		// Check if the space under the brick is completely empty
		cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
		cPos.Y--
		if cPos.Y < 0 {
			// The brick is already at the bottom, it cannot fall any lower.
			continue
		}

		// Get the rectangle below the brick.
		pPos := w.CanonicalPosToPixelsPos(cPos)
		r := w.BrickBounds(pPos)

		// Check if anything is in the rectangle.
		obstacles := w.GetObstacles(b)
		if RectIntersectsRects(r, obstacles) {
			// There's something in the space.
			continue
		}

		b.Falling = true
		b.FallingSpeed = 0
	}
}

func (w *World) AdvanceFallingBricks() {
	for i := range w.Bricks {
		b := &w.Bricks[i]
		if !b.Falling {
			// Skip non-falling bricks.
			continue
		}

		// Move the brick.
		r := w.BrickBounds(b.PixelPos)
		obstacles := w.GetObstacles(b)
		b.FallingSpeed += w.BrickFallAcceleration
		newR, nPixelsLeft := MoveRect(r, r.Corner1.Plus(Pt{0, 1000}),
			b.FallingSpeed, obstacles)
		b.PixelPos = newR.Corner1

		if nPixelsLeft > 0 {
			// We hit something.
			// Mark the brick as no longer falling and move it to its canonical
			// position.
			b.Falling = false
			b.FallingSpeed = 0
			cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
			b.PixelPos = w.CanonicalPosToPixelsPos(cPos)
		}
	}
}

func (w *World) ComputeDraggedBrickPosition(input PlayerInput) Pt {
	targetPos := input.Pos.Plus(w.DraggingOffset)
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
	obstacles := w.GetObstacles(w.Dragging)
	brick := w.BrickBounds(w.Dragging.PixelPos)

	nMaxPixels := 100

	// First, go as far as possible towards the target, in a straight line.
	brick, nMaxPixels = MoveRect(brick, targetPos, nMaxPixels, obstacles)

	// Now, go towards the target's X as much as possible.
	brick, nMaxPixels = MoveRect(brick, Pt{targetPos.X, brick.Corner1.Y},
		nMaxPixels, obstacles)

	// Now, go towards the target's X as much as possible.
	brick, nMaxPixels = MoveRect(brick, Pt{brick.Corner1.X, targetPos.Y},
		nMaxPixels, obstacles)

	return brick.Corner1
}

func (w *World) BrickBounds(posPixels Pt) (r Rectangle) {
	r.Corner1 = posPixels
	r.Corner2 = posPixels
	r.Corner2.X += w.BrickPixelSize
	r.Corner2.Y += w.BrickPixelSize
	return
}

func (w *World) GetObstacles(exception *Brick) (obstacles []Rectangle) {
	obstacles = make([]Rectangle, 0, len(w.Bricks))
	for j := range w.Bricks {
		if exception == &w.Bricks[j] {
			continue
		}
		obstacles = append(obstacles, w.BrickBounds(w.Bricks[j].PixelPos))
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
	return
}
