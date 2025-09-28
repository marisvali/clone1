package main

import (
	"fmt"
	"math"
)

// World rules (physics)
// ---------------------
//
// Terminology:
// - Slot: the space of the world is a matrix of 6x8 rectangles separated by
// some margins. These rectangular places are called slots. Some are empty, some
// will have bricks in them.
// - Brick: rectangular objects that fit inside slots that the player can drag
// around and match.
// - Canonical position: a position in the matrix of slots. For example
// (2, 0) means the third brick on the bottom row, going left to right.
// - Pixel position: the position of a brick in terms of pixels, not slots. The
// pixel position can place a brick between slots.
// - Canonical pixel position: a position in the matrix of slots converted to
// pixel coordinates. For example depending on brick size and margin sizes the
// canonical position (2, 0) can be turned into the canonical pixel position
// (230, 30).
// - Canonical brick: a brick that just stays in its slot. It is not falling and
// it is not dragged around by the player. This is the default state for bricks.
// - Dragged brick: a brick that the player has clicked on and is currently
// dragging around the world space.
// - Falling brick: a brick that is not dragged by the player and is currently
// falling because it has nothing underneath it.
//
// The behavior of a canonical brick:
// - If the brick's pixel position is a canonical pixel position, it doesn't
// move. Otherwise, it moves towards the nearest canonical pixel position.
// - If the slot underneath it is completely empty, it becomes a falling brick.
// Completely empty means there's no part of another brick (dragged, canonical
// or falling) in the slot.
// - If the player clicks it, it becomes a dragged brick.
//
// The behavior of a dragged brick:
// - When the player releases the click while dragging a brick, the dragged
// brick becomes canonical. The behavior of the canonical brick will then take
// care of putting the brick in the right place.
// - When the player moves the mouse while dragging a brick, the brick moves
// towards the mouse position.
// - If the dragged brick hits a wall or another brick, it stops moving on the
// axis where it hit the obstacle, but it continues moving on the other axis
// as long as possible.
// - The dragged brick goes towards the mouse cursor with some limited speed,
// it doesn't just teleport to the nearest valid position in a single frame.
//
// The behavior of a falling brick:
// - A falling brick moves down each frame with a limited speed, with
// acceleration.
// - A falling brick becomes canonical when it intersects another brick or the
// bottom of the playable region. The behavior of the canonical brick will then
// take care of putting the brick in the right place.
// - A falling brick becomes dragged if the player clicks on it.
//
// One important element for making the behavior of the world as bug-free as
// possible is to never 'teleport' bricks by setting their positions directly.
// Always move them towards a position. This ensures that bricks will never
// overlap.

type BrickState int64

const (
	Canonical BrickState = iota
	Dragged
	Falling
)

type Brick struct {
	Val          int64
	PixelPos     Pt
	State        BrickState
	FallingSpeed int64
}

type World struct {
	NCols                 int64
	NRows                 int64
	BrickPixelSize        int64
	MarginPixelSize       int64
	BrickFallAcceleration int64
	Bricks                []Brick
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

	for y := int64(0); y < 4; y++ {
		for x := int64(0); x < 6; x++ {
			w.Bricks = append(w.Bricks, Brick{
				Val:      (x+y)%3 + 1,
				PixelPos: w.CanonicalPosToPixelsPos(Pt{x, y}),
				State:    Canonical,
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
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelsPosToCanonicalPos(pixelPos Pt) (matPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	matPos.X = int64(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	matPos.Y = int64(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) CanonicalPosToPixelsPos(matPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = matPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (matPos.Y+1)*l
	return
}

func (w *World) Step(input PlayerInput) {
	w.UpdateDraggedBrick(input)
	w.UpdateFallingBricks()
	w.UpdateCanonicalBricks()

	// Check if bricks intersect each other or are out of bounds.
	{
		for i := range w.Bricks {
			obstacles := w.GetObstacles(&w.Bricks[i])
			brick := w.BrickBounds(w.Bricks[i].PixelPos)
			// Don't use RectIntersectsRects because I want to be able to
			// put a breakpoint here and see which rect intersects which.
			for j := range obstacles {
				if brick.Intersects(obstacles[j]) {
					Check(fmt.Errorf("solids intersect each other"))
				}
			}
		}
	}
}

func (w *World) UpdateDraggedBrick(input PlayerInput) {
	var dragged *Brick
	for i := range w.Bricks {
		if w.Bricks[i].State == Dragged {
			// It should not be possible to be dragging anything already.
			if dragged != nil {
				Check(fmt.Errorf("started dragging a brick while another " +
					"brick was already marked as dragging"))
			}
			dragged = &w.Bricks[i]
		}
	}

	if input.JustPressed {
		// Check if there's any brick under the click.
		for i := range w.Bricks {
			p := w.Bricks[i].PixelPos
			brickSize := Pt{w.BrickPixelSize, w.BrickPixelSize}
			r := Rectangle{p, p.Plus(brickSize)}
			if r.ContainsPt(input.Pos) {
				w.Bricks[i].State = Dragged
				dragged = &w.Bricks[i]
				w.DraggingOffset = p.Minus(input.Pos)
				break
			}
		}
	}

	if input.JustReleased {
		if dragged != nil {
			dragged.State = Canonical
		}
	}

	if dragged == nil {
		return
	}

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
	obstacles := w.GetObstacles(dragged)
	brick := w.BrickBounds(dragged.PixelPos)

	nMaxPixels := int64(100)

	// First, go as far as possible towards the target, in a straight line.
	brick, nMaxPixels = MoveRect(brick, targetPos, nMaxPixels, obstacles)

	// Now, go towards the target's X as much as possible.
	brick, nMaxPixels = MoveRect(brick, Pt{targetPos.X, brick.Corner1.Y},
		nMaxPixels, obstacles)

	// Now, go towards the target's X as much as possible.
	brick, nMaxPixels = MoveRect(brick, Pt{brick.Corner1.X, targetPos.Y},
		nMaxPixels, obstacles)

	dragged.PixelPos = brick.Corner1
}

func (w *World) UpdateFallingBricks() {
	for i := range w.Bricks {
		b := &w.Bricks[i]
		if b.State != Falling {
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
			// The brick becomes canonical.
			b.State = Canonical
			b.FallingSpeed = 0
		}
	}
}

func (w *World) UpdateCanonicalBricks() {
	// Mark falling bricks.
	for i := range w.Bricks {
		b := &w.Bricks[i]

		// Skip non-canonical bricks.
		if b.State != Canonical {
			continue
		}

		// Check if the brick is at canonical position and the space under the
		// brick is completely empty. It is important to check if the brick
		// is at the canonical position before going to falling. When the player
		// releases a dragging brick, we want the released brick to move to a
		// canonical position before starting to fall.
		if w.AtCanonicalPosition(b) && w.SpaceUnderBrickIsEmpty(b) {
			b.State = Falling
			b.FallingSpeed = 0
		} else {
			// Go towards the closest canonical pos.
			cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
			pPos := w.CanonicalPosToPixelsPos(cPos)

			// Move the brick.
			r := w.BrickBounds(b.PixelPos)
			obstacles := w.GetObstacles(b)
			newR, _ := MoveRect(r, pPos, 20, obstacles)
			b.PixelPos = newR.Corner1
		}
	}
}

func (w *World) AtCanonicalPosition(b *Brick) bool {
	cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
	pPos := w.CanonicalPosToPixelsPos(cPos)
	return pPos == b.PixelPos
}

func (w *World) SpaceUnderBrickIsEmpty(b *Brick) bool {
	cPos := w.PixelsPosToCanonicalPos(b.PixelPos)
	cPos.Y--
	if cPos.Y < 0 {
		// The brick is already at the bottom, it cannot fall any lower.
		return false
	}

	// Get the rectangle below the brick.
	pPos := w.CanonicalPosToPixelsPos(cPos)
	r := w.BrickBounds(pPos)

	// Check if anything is in the rectangle.
	obstacles := w.GetObstacles(b)
	if RectIntersectsRects(r, obstacles) {
		// There's something in the space.
		return false
	}
	return true
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
