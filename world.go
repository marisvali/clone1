package main

import (
	"fmt"
	"math"
	"math/rand"
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

type WorldState int64

const (
	Regular WorldState = iota
	ComingUp
	Lost
	Won
)

type World struct {
	NCols                 int64
	NRows                 int64
	BrickPixelSize        int64
	MarginPixelSize       int64
	BrickFallAcceleration int64
	Bricks                []Brick
	DraggingOffset        Pt
	DebugPts              []Pt
	RegularCooldown       int64
	RegularCooldownIdx    int64
	ComingUpDistanceLeft  int64
	ComingUpSpeed         int64
	ComingUpDeceleration  int64
	State                 WorldState
	PreviousState         WorldState
	SolvedFirstState      bool
}

type PlayerInput struct {
	Pos             Pt
	JustPressed     bool
	JustReleased    bool
	ResetWorld      bool
	TriggerComingUp bool
}

func (w *World) Initialize() {
	w.NCols = 6
	w.NRows = 8
	w.MarginPixelSize = 30
	w.BrickPixelSize = (playWidth - (w.MarginPixelSize * (w.NCols + 1))) / w.NCols
	w.BrickFallAcceleration = 2
	w.ComingUpDeceleration = 2
	w.RegularCooldown = 20
	w.RegularCooldownIdx = w.RegularCooldown

	w.Bricks = []Brick{}
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
}

func NewWorld() (w World) {
	w.Initialize()
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelsPosToCanonicalPos(pixelPos Pt) (canPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	canPos.X = int64(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	canPos.Y = int64(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) CanonicalPosToPixelsPos(canPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = canPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (canPos.Y+1)*l
	return
}

func (w *World) PixelPosToCanonicalPixelPos(pixelPos Pt) (canPixelPos Pt) {
	canPos := w.PixelsPosToCanonicalPos(pixelPos)
	canPixelPos = w.CanonicalPosToPixelsPos(canPos)
	return
}

func (w *World) StepRegular(justEnteredState bool, input PlayerInput) {
	if justEnteredState {
		w.RegularCooldownIdx = w.RegularCooldown
	}
	// w.RegularCooldownIdx--
	// if w.RegularCooldownIdx == 0 {
	// 	w.State = ComingUp
	// 	return
	// }

	w.UpdateDraggedBrick(input)
	w.UpdateFallingBricks()
	w.UpdateCanonicalBricks()
	w.MergeBricks()
}

func (w *World) StepComingUp(justEnteredState bool) {
	if justEnteredState {
		// We have to compute the speed we need to start with in order to
		// decelerate by the desired deceleration rate and travel the desired
		// distance in the desired time and reach the destination with speed
		// zero or close to zero.
		// In order to do this, reverse the problem: if we start with speed 0
		// and keep increasing the speed, what speed to we reach by the time we
		// cover the distance?
		totalDist := w.BrickPixelSize + w.MarginPixelSize
		distSoFar := int64(0)
		speed := int64(0)
		acc := w.ComingUpDeceleration
		requiredSteps := 0
		for distSoFar < totalDist {
			speed += acc
			distSoFar += speed
			requiredSteps++
		}

		// We set this starting speed. We know that we will travel the total
		// distance when we reach speed 0 or right before.
		w.ComingUpSpeed = speed
		w.ComingUpDistanceLeft = w.BrickPixelSize + w.MarginPixelSize

		// Create a new row of bricks.
		for x := range w.NCols {
			w.Bricks = append(w.Bricks, Brick{
				Val:      int64(rand.Intn(3)) + 1,
				PixelPos: w.CanonicalPosToPixelsPos(Pt{x, -1}),
				State:    Canonical,
			})
		}
	}

	// In the last step, the speed might be higher than the distance left.
	// In this case, just travel the exact distance left.
	if w.ComingUpSpeed > w.ComingUpDistanceLeft {
		w.ComingUpSpeed = w.ComingUpDistanceLeft
	}
	for i := range w.Bricks {
		w.Bricks[i].PixelPos.Y -= w.ComingUpSpeed
	}
	w.ComingUpDistanceLeft -= w.ComingUpSpeed
	w.ComingUpSpeed -= w.ComingUpDeceleration

	// Check if we're done.
	if w.ComingUpDistanceLeft == 0 {
		// Check if bricks went over the top.
		for i := range w.Bricks {
			b := &w.Bricks[i]
			bottom := playHeight - w.MarginPixelSize
			top := bottom - w.BrickPixelSize*w.NRows - w.MarginPixelSize*(w.NRows-1)
			brickTop := w.BrickBounds(w.Bricks[i].PixelPos).Corner1.Y

			if brickTop >= top {
				// The brick is not over the top.
				continue
			}

			// Brick is over the top. If it's not a Dragged brick, the game is
			// over.
			if w.Bricks[i].State != Dragged {
				w.State = Lost
				return
			}

			// The dragged brick is moved over the top. Try to move it down so
			// that it's not over the top anymore.
			r := w.BrickBounds(b.PixelPos)
			obstacles := w.GetObstacles(b, WithoutTop)
			newR, nPixelsLeft := MoveRect(r, r.Corner1.Plus(Pt{0, 1000}),
				top-brickTop, obstacles)
			b.PixelPos = newR.Corner1

			if nPixelsLeft > 0 {
				// We couldn't move the brick all the way down, which means it
				// hit another brick, so it's game over.
				w.State = Lost
				return
			}
		}
		w.State = Regular
		return
	}
}

func (w *World) Step(input PlayerInput) {
	// Reset the world.
	if input.ResetWorld {
		w.Initialize()
	}

	// Trigger a coming up event.
	if input.TriggerComingUp {
		w.State = ComingUp
	}

	var justEnteredState bool
	if !w.SolvedFirstState {
		justEnteredState = true
		w.SolvedFirstState = true
	} else {
		justEnteredState = w.State != w.PreviousState
		w.PreviousState = w.State
	}

	switch w.State {
	case Regular:
		w.StepRegular(justEnteredState, input)
	case ComingUp:
		w.StepComingUp(justEnteredState)
	}

	// Check if bricks intersect each other or are out of bounds.
	{
		for i := range w.Bricks {
			obstacles := w.GetObstacles(&w.Bricks[i], WithTop)
			brick := w.BrickBounds(w.Bricks[i].PixelPos)
			// Don't use RectIntersectsRects because I want to be able to
			// put a breakpoint here and see which rect intersects which.
			for j := range obstacles {
				if brick.Intersects(obstacles[j]) {
					// Check(fmt.Errorf("solids intersect each other"))
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
	obstacles := w.GetObstacles(dragged, WithTop)
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
		obstacles := w.GetObstacles(b, WithTop)
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
			canPixelPos := w.PixelPosToCanonicalPixelPos(b.PixelPos)

			// Move the brick.
			r := w.BrickBounds(b.PixelPos)
			obstacles := w.GetObstacles(b, WithTop)
			newR, _ := MoveRect(r, canPixelPos, 20, obstacles)
			b.PixelPos = newR.Corner1
		}
	}
}

func (w *World) AtCanonicalPosition(b *Brick) bool {
	canPixelPos := w.PixelPosToCanonicalPixelPos(b.PixelPos)
	return canPixelPos == b.PixelPos
}

func (w *World) SpaceUnderBrickIsEmpty(b *Brick) bool {
	canPos := w.PixelsPosToCanonicalPos(b.PixelPos)
	canPos.Y--
	if canPos.Y < 0 {
		// The brick is already at the bottom, it cannot fall any lower.
		return false
	}

	// Get the rectangle below the brick.
	canPixelPos := w.CanonicalPosToPixelsPos(canPos)
	r := w.BrickBounds(canPixelPos)

	// Check if anything is in the rectangle.
	obstacles := w.GetObstacles(b, WithTop)
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

type GetObstaclesOption int64

const (
	WithTop GetObstaclesOption = iota
	WithoutTop
)

func (w *World) GetObstacles(exception *Brick,
	o GetObstaclesOption) (obstacles []Rectangle) {
	obstacles = make([]Rectangle, 0, len(w.Bricks))
	for j := range w.Bricks {
		if exception == &w.Bricks[j] {
			continue
		}
		// Skip bricks that have the same value.
		if exception.Val == w.Bricks[j].Val {
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
	if o == WithTop {
		obstacles = append(obstacles, topRect)
	}
	obstacles = append(obstacles, leftRect)
	obstacles = append(obstacles, rightRect)
	return
}

func (w *World) MergeBricks() {
	// Keep doing merges until no merges are possible anymore.
	// I don't expect to ever have more than one merge happen in one frame but
	// I feel weird hardcoding that assumption when I can just add a loop to
	// handle that case as well.
	for {
		foundMerge, i, j := w.FindMergingBricks()
		if !foundMerge {
			return
		}

		// A merge occurred. A brick will disappear and one will have
		// its value increased.
		// How do I choose which one disappears and which one has its
		// value increased?
		// The most common case is that the dragged brick is dragged on
		// top of a canonical brick. It's also common for a falling
		// brick to get on top of a canonical brick. Less common, but
		// possible, is to have a falling brick get on top of a dragged
		// brick. Something that can happen more often that it seems likely,
		// a canonical brick and move on top of a canonical brick. This is
		// because the player drags a brick near the one they intend to merge
		// with and then releases the brick early. The released brick becomes
		// canonical and is now moving towards the position where the static
		// brick is.
		// The way to cover all these cases in one is to detect which of
		// the two bricks is closer to a canonical position. That one
		// gets its value increased, the other one disappears. And the
		// one which gets its value increased becomes a canonical brick,
		// just to cover any weird edge cases. I feel like the result
		// of a merge should go to a canonical position first and if it
		// then needs to fall, it does so after it goes to the canonical
		// position.
		b1 := &w.Bricks[i]
		b2 := &w.Bricks[j]
		canPos1 := w.PixelPosToCanonicalPixelPos(b1.PixelPos)
		dif1 := b1.PixelPos.SquaredDistTo(canPos1)

		canPos2 := w.PixelPosToCanonicalPixelPos(b2.PixelPos)
		dif2 := b2.PixelPos.SquaredDistTo(canPos2)

		var idxToRemove int
		var brickToUpdate *Brick
		if dif1 < dif2 {
			// b1 is closer to a canonical pos.
			brickToUpdate = b1
			idxToRemove = j
		} else {
			// b2 is closer to a canonical pos.
			brickToUpdate = b2
			idxToRemove = i
		}

		brickToUpdate.Val++

		// Do a loop for now between values as I don't have all the
		// values and the rules for them are not yet clear.
		if brickToUpdate.Val > 3 {
			brickToUpdate.Val = 1
		}
		brickToUpdate.State = Canonical

		// Remove from slice efficiently.
		w.Bricks[idxToRemove] = w.Bricks[len(w.Bricks)-1]
		w.Bricks = w.Bricks[:len(w.Bricks)-1]
	}
}

func (w *World) FindMergingBricks() (foundMerge bool, i, j int) {
	for i = range w.Bricks {
		for j = range w.Bricks {
			if i == j {
				continue
			}

			dist := w.Bricks[i].PixelPos.SquaredDistTo(w.Bricks[j].PixelPos)
			// Two bricks merge if they are close enough for each other.
			// We decide here what "close enough" means.
			if dist < Sqr(w.BrickPixelSize/3) {
				return true, i, j
			}
		}
	}
	return false, 0, 0
}
