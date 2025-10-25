package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
)

// World rules (physics)
// ---------------------
//
// Terminology
// -----------
//
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
// High-level approach
// -------------------
//
// We know that in general we want to drag bricks around, have them bump against
// each other if they are different values, have them merge if they have the
// same value, have new bricks come up, have bricks fall down if they have
// nothing underneath.
//
// Once we know largely what effect we want to obtain, we can implement it in
// several ways. The challenge is then to avoid errors in edge cases. For
// example what if a brick is dragged over a falling brick with the same value,
// or bricks are coming up right as another brick is falling etc.
//
// At first it seemed that one simple rule is to avoid having bricks intersect
// each other. And one simple way to ensure this is to never 'teleport' a brick,
// as in, set its position directly. Instead, move every brick by using a
// function that checks for collisions and stops movement before an intersection
// occurs. This way, we simply never enter an erroneous state. Of course we need
// a special exception for bricks that have the same value and move on top of
// each other. They do not merge immediately when they start intersecting, so we
// must allow intersection between bricks of the same value.
//
// I found that this approach cannot work. The merging mechanics cause at least
// one edge case where I want to allow an erroneous state to occur:
// - Let's say I have 3 bricks of the same value: A, B and C.
// - A is sitting.
// - B is dragged by the player and intersects A on the left or right side.
// - C is falling on top of A.
// When C hits A, they merge and A gets a different value than B. Suddenly, two
// bricks with different values are overlapping. The issue is that I do not want
// to prevent this from happening, because it would mean introducing some rule
// that would make the world act strangely. I could say that a brick may not
// intersect more than one other brick with the same value, at any one time. But
// then I can find edge cases where that would be annoying.
//
// I want to preserve the effect that bricks of the same value can always go
// through each other. But when they merge, bricks change their value. This
// inevitably leads to cases where a previously valid intersection becomes
// invalid.
//
// The conclusion is that a better approach is to have a system that tolerates
// invalid intersections and solves them as soon as possible, in a way that
// feels natural. In the end the system was implemented in the behavior of
// canonical bricks.

// The behavior of canonical bricks
// --------------------------------
//
// - For each canonical brick, we compute its target position. Normally this
// means the nearest canonical pixel position. However, there will be edge cases
// where two canonical bricks will have the same nearest canonical pixel
// position. In those cases, an algorithm decides which brick gets which
// position. The algorithm is described in more detail in the implementation.
// - If the brick's pixel position is a canonical pixel position, it doesn't
// move. Otherwise, it moves towards the nearest canonical pixel position,
// disregarding any intersections.
// - If the slot underneath it is completely empty, it becomes a falling brick.
// Completely empty means there's no part of another brick (dragged, canonical
// or falling) in the slot.
// - If the player clicks it, it becomes a dragged brick.
//
// The behavior of a dragged brick
// -------------------------------
//
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
// - The dragged brick doesn't intersect other bricks most of the time, because
// it stops moving when it hits an obstacle. But it can find itself overlapping
// a brick that used to be of the same value, but which suddenly just changed
// value due to a merge. To cover this case, the rule is that a dragged brick
// that intersects another brick becomes canonical and the player loses control
// of it.
//
// The behavior of a falling brick
// -------------------------------
//
// - A falling brick moves down each frame with a limited speed, with
// acceleration.
// - A falling brick becomes canonical when it intersects another brick or the
// bottom of the playable region. The behavior of the canonical brick will then
// take care of putting the brick in the right place.
// - A falling brick becomes dragged if the player clicks on it.
//
// Note
// ----
//
// One interesting note: at first, it seemed natural to implement a behavior
// for each brick type, and have these behaviors generate the desired global
// effect automatically. It seemed like a natural fit for a Brick interface
// with a different implementation for a Step/Update function of each brick
// type. However, for canonical bricks, it proved very useful to have a global
// algorithm that solves conflicts between canonical bricks. This would have
// been very difficult to achieve if each canonical brick would decide its
// behavior for itself in isolation. So having a global function that updates
// all canonical bricks, and another one that updates all falling bricks for
// example allowed for a flexibility that paid off significantly.

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
	// Derived values.
	CanonicalPos      Pt
	CanonicalPixelPos Pt
	Bounds            Rectangle
}

func ChangePixelPos(b *Brick, w *World, newPos Pt) {
	b.PixelPos = newPos
	b.Bounds = w.BrickBounds(b.PixelPos)
	b.CanonicalPos = w.PixelPosToCanonicalPos(b.PixelPos)
	b.CanonicalPixelPos = w.CanonicalPosToPixelPos(b.CanonicalPos)
}

type WorldState int64

const (
	Regular WorldState = iota
	ComingUp
	Lost
	Won
)

type World struct {
	Rand
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
	AssertionFailed       bool
	MaxBrickValue         int64
	ObstaclesBuffer       []Rectangle
	ColumnsBuffer         [][]*Brick
	CanPosBuffer          []Pt
}

type PlayerInput struct {
	Pos             Pt
	JustPressed     bool
	JustReleased    bool
	ResetWorld      bool
	TriggerComingUp bool
}

func (w *World) Initialize() {
	w.RSeed(0)
	w.NCols = 6
	w.NRows = 8
	w.MaxBrickValue = 10
	w.MarginPixelSize = 30
	w.BrickPixelSize = (playWidth - (w.MarginPixelSize * (w.NCols + 1))) / w.NCols
	w.BrickFallAcceleration = 2
	w.ComingUpDeceleration = 2
	w.RegularCooldown = 20
	w.RegularCooldownIdx = w.RegularCooldown
	w.ObstaclesBuffer = make([]Rectangle, w.NCols*w.NRows+4)
	w.ColumnsBuffer = make([][]*Brick, w.NCols)
	for i := range w.ColumnsBuffer {
		w.ColumnsBuffer[i] = make([]*Brick, w.NRows)
	}
	w.CanPosBuffer = make([]Pt, w.NCols*w.NRows)

	w.Bricks = make([]Brick, 0, w.NRows*w.NCols)
	for y := int64(0); y < 4; y++ {
		for x := int64(0); x < 6; x++ {
			w.AddBrick(Pt{x, y}, (x*y)%w.MaxBrickValue+1)
		}
	}
	// Test edge case with 4 bricks of the same value.
	// w.Bricks = []Brick{}
	// w.Bricks = append(w.Bricks, Brick{
	// 	Val:      1,
	// 	PixelPos: w.CanonicalPosToPixelPos(Pt{0, 0}),
	// 	State:    Canonical,
	// })
	// w.Bricks = append(w.Bricks, Brick{
	// 	Val:      1,
	// 	PixelPos: w.CanonicalPosToPixelPos(Pt{1, 0}),
	// 	State:    Canonical,
	// })
	// w.Bricks = append(w.Bricks, Brick{
	// 	Val:      1,
	// 	PixelPos: w.CanonicalPosToPixelPos(Pt{2, 0}),
	// 	State:    Canonical,
	// })
	// w.Bricks = append(w.Bricks, Brick{
	// 	Val:      1,
	// 	PixelPos: w.CanonicalPosToPixelPos(Pt{0, 7}),
	// 	State:    Canonical,
	// })
	// Test edge case with 3 bricks of the same value.
	// for y := 0; y < 1; y++ {
	// 	for x := 0; x < 3; x++ {
	// 		w.Bricks = append(w.Bricks, Brick{
	// 			Val: 3,
	// 			// PosMat:    Pt{x, y},
	// 			PixelPos: w.CanonicalPosToPixelPos(Pt{x, y}),
	// 		})
	// 	}
	// }
}

func (w *World) AddBrick(canPos Pt, val int64) {
	w.Bricks = append(w.Bricks, Brick{
		Val:   val,
		State: Canonical,
	})

	ChangePixelPos(&w.Bricks[len(w.Bricks)-1], w, w.CanonicalPosToPixelPos(canPos))
}

func NewWorld() (w World) {
	w.Initialize()
	return w
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

	// The test for game over is currently in StepComingUp.
	// Consider testing for game over here, as well, or inside StepRegular, just
	// as an added precaution, even if I can't think of a way in which a game
	// over could be reached during a StepRegular.
}

func (w *World) StepRegular(justEnteredState bool, input PlayerInput) {
	if justEnteredState {
		w.RegularCooldownIdx = w.RegularCooldown
	}

	// TODO: change this
	// Disable for now the regular event of bricks coming up, to allow easier
	// testing during development.
	// w.RegularCooldownIdx--
	// if w.RegularCooldownIdx == 0 {
	// 	w.State = ComingUp
	// 	return
	// }

	w.UpdateDraggedBrick(input)
	w.UpdateFallingBricks()
	w.UpdateCanonicalBricks()
	w.MergeBricks()

	// Disable the check below as we currently do allow intersections to occur
	// in some cases and the strategy is to recover from them. So "solids
	// should never intersect" is no longer a valid invariant to check against.
	//
	// Check if bricks intersect each other or are out of bounds.
	// {
	// 	for i := range w.Bricks {
	// 		obstacles := w.GetObstacles(&w.Bricks[i], IncludingTop)
	// 		brick := w.BrickBounds(w.Bricks[i].PixelPos)
	// 		// Don't use RectIntersectsRects because I want to be able to
	// 		// put a breakpoint here and see which rect intersects which.
	// 		for j := range obstacles {
	// 			if brick.Intersects(obstacles[j]) {
	// 				// Check(fmt.Errorf("solids intersect each other"))
	// 				w.AssertionFailed = true
	// 			}
	// 		}
	// 	}
	// }
}

func (w *World) UpdateDraggedBrick(input PlayerInput) {
	var dragged *Brick
	for i := range w.Bricks {
		if w.Bricks[i].State == Dragged {
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
				// We were dragging a brick but we just clicked on another
				// brick to start dragging it? This should not be possible.
				if dragged != nil {
					Check(fmt.Errorf("started dragging a brick while another " +
						"brick was already marked as dragging"))
				}

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
			return
		}
	}

	if dragged == nil {
		return
	}

	// Get the set of rectangles the brick must not intersect.
	obstacles := w.GetObstacles(dragged, IncludingTop)

	// If the dragged brick intersects something, it becomes canonical and the
	// behavior of canonical bricks will resolve the intersection.
	if RectIntersectsRects(dragged.Bounds, obstacles) {
		dragged.State = Canonical
		return
	}

	targetPos := input.Pos.Plus(w.DraggingOffset)
	nMaxPixels := int64(100)
	w.MoveBrick(dragged, targetPos, nMaxPixels, SlideOnObstacles)
}

func (w *World) UpdateFallingBricks() {
	for i := range w.Bricks {
		b := &w.Bricks[i]
		if b.State != Falling {
			// Skip non-falling bricks.
			continue
		}

		// Move the brick.
		b.FallingSpeed += w.BrickFallAcceleration
		hitObstacle := w.MoveBrick(b, b.PixelPos.Plus(Pt{0, 1000}),
			b.FallingSpeed, StopAtFirstObstacleIncludingTop)
		if hitObstacle {
			// We hit something.
			// The brick becomes canonical.
			b.State = Canonical
			b.FallingSpeed = 0
		}
	}
}

func (w *World) MarkFallingBricks() {
	for i := range w.Bricks {
		b := &w.Bricks[i]

		// Skip non-canonical bricks.
		if b.State != Canonical {
			continue
		}

		if b.PixelPos != b.CanonicalPixelPos {
			// Skip bricks which are not at their canonical position.
			continue
		}

		// Check if there's anything under this brick.
		canPosUnder := b.CanonicalPos
		canPosUnder.Y--
		if canPosUnder.Y < 0 {
			// The brick is already at the bottom, it cannot fall any lower.
			continue
		}

		slotUnderneath := w.BrickBounds(w.CanonicalPosToPixelPos(canPosUnder))

		// Check if any bricks intersect slotUnderneath.
		intersects := false
		for j := range w.Bricks {
			otherB := &w.Bricks[j]
			if i != j && b.Val != otherB.Val {
				// possible intersection, check
				if slotUnderneath.Intersects(otherB.Bounds) {
					intersects = true
					break
				}
			}
		}
		if !intersects {
			b.State = Falling
			b.FallingSpeed = 0
		}
	}
}

func (w *World) UpdateCanonicalBricks() {
	w.MarkFallingBricks()

	// Decide the target position for each canonical brick:
	// - Assign each brick to a column. Usually canonical bricks are firmly in
	// a column or another. But a dragged brick becomes canonical when released
	// and it can be released in any position. So we may always have at least
	// one canonical brick in some non-standard position, e.g. between two
	// columns. But, even if a brick is between two columns, it is closer to one
	// than another.
	// - For the bricks in a column, decide which goes into what position. The
	// easiest way to do this is to get the bottom one first, decide that one
	// cannot move any lower, so it gets the bottom position. The next one must
	// necessarily get the next available position, on top of the first one,
	// and so on.
	// - This may result in some bricks moving up in order to fit well with the
	// others. But normally they will travel a short distance and it should look
	// natural to the player.
	// - An exception has to be made for bricks that have the same value. If two
	// bricks with the same value compete for the same spot, they are allowed to
	// go for it. This is because they are competing because they are probably
	// already overlapping significantly, which means a merge is imminent. It
	// looks a little ridiculous if they go on top of each other, then one falls
	// on the other and they merge.
	//
	// By following this algorithm, we guarantee that bricks end up in valid
	// positions and any intersections get solved relatively quickly in a way
	// that feels natural.
	//
	// Assign each brick to a column.
	columns := w.ColumnsBuffer
	for i := range columns {
		columns[i] = w.ColumnsBuffer[i][:0]
	}

	for i := range w.Bricks {
		b := &w.Bricks[i]

		// Skip non-canonical bricks.
		if b.State != Canonical {
			continue
		}

		// Possible assert: the column is valid.
		columns[b.CanonicalPos.X] = append(columns[b.CanonicalPos.X], b)
	}

	// Go column by column.
	for _, column := range columns {
		// Sort bricks in the column by their Y position, so that we can iterate
		// through bricks from bottom to top.
		// sort.Slice(column, func(i, j int) bool {
		// 	return column[i].PixelPos.Y > column[j].PixelPos.Y
		// })
		slices.SortFunc(column, func(b1, b2 *Brick) int {
			return cmp.Compare(b1.PixelPos.Y, b2.PixelPos.Y)
		})

		lastTargetCanPos := Pt{-1000, -1000}
		for i := range column {
			b := column[i]
			// Get target pos.
			targetCanPos := b.CanonicalPos
			// If it intersects with an already decided target pos, go to the
			// next available canonical target pos. However, we are going from
			// bottom to top so the only thing it can intersect with is the
			// previous target pos, and the higher target pos is definitely
			// available.
			if targetCanPos == lastTargetCanPos {
				// Only put the brick at a higher pos if it has a different
				// value than the brick at the current pos. Otherwise, we are
				// dealing with two bricks of the same value which are
				// overlapping but not yet merged.
				// In this case, just let the targetCanPos be the same as the
				// lastTargetCanPos. This will move the current brick towards
				// the existing brick and they will soon merge.
				// If for any reason any of the two bricks changes value in a
				// future frame (though I can't currently imagine a case where
				// this happens), then the algorithm will run again and drive
				// these bricks apart.
				if i > 0 && column[i].Val != column[i-1].Val {
					targetCanPos.Y += 1
				}
			}
			lastTargetCanPos = targetCanPos
			targetPos := b.CanonicalPixelPos

			// Go towards the target pos, without considering any obstacles.
			w.MoveBrick(b, targetPos, 21, IgnoreObstacles)
		}
	}
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
		// a canonical brick moves on top of a canonical brick. This is because
		// the player drags a brick near the one they intend to merge with and
		// then releases the brick early. The released brick becomes canonical
		// and is now moving towards the position where the static brick is.
		//
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

		// TODO: change this
		// Do a loop for now between values as I don't have all the
		// values and the rules for them are not yet clear.
		if brickToUpdate.Val > w.MaxBrickValue {
			brickToUpdate.Val = 1
		}
		brickToUpdate.State = Canonical

		// Remove from slice efficiently.
		w.Bricks[idxToRemove] = w.Bricks[len(w.Bricks)-1]
		w.Bricks = w.Bricks[:len(w.Bricks)-1]
	}
}

func (w *World) FindMergingBricks() (foundMerge bool, i, j int) {
	// Two bricks merge if they are close enough for each other.
	// We decide here what "close enough" means.
	mergeDist := Sqr(w.BrickPixelSize / 3)
	for i = range w.Bricks {
		for j = i + 1; j < len(w.Bricks); j++ {
			if w.Bricks[i].Val != w.Bricks[j].Val {
				continue
			}

			dist := w.Bricks[i].PixelPos.SquaredDistTo(w.Bricks[j].PixelPos)
			if dist < mergeDist {
				return true, i, j
			}
		}
	}
	return false, 0, 0
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
			w.AddBrick(Pt{x, -1}, w.RInt(1, w.MaxBrickValue))
		}
	}

	// In the last step, the speed might be higher than the distance left.
	// In this case, just travel the exact distance left.
	if w.ComingUpSpeed > w.ComingUpDistanceLeft {
		w.ComingUpSpeed = w.ComingUpDistanceLeft
	}
	for i := range w.Bricks {
		newPos := w.Bricks[i].PixelPos
		newPos.Y -= w.ComingUpSpeed
		ChangePixelPos(&w.Bricks[i], w, newPos)
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
			brickTop := w.Bricks[i].Bounds.Corner1.Y

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
			hitObstacle := w.MoveBrick(b, b.PixelPos.Plus(Pt{0, 1000}),
				top-brickTop, StopAtFirstObstacleExceptTop)

			if hitObstacle {
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

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelPosToCanonicalPos(pixelPos Pt) (canPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	canPos.X = int64(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	canPos.Y = int64(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) CanonicalPosToPixelPos(canPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = canPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (canPos.Y+1)*l
	return
}

func (w *World) PixelPosToCanonicalPixelPos(pixelPos Pt) (canPixelPos Pt) {
	canPos := w.PixelPosToCanonicalPos(pixelPos)
	canPixelPos = w.CanonicalPosToPixelPos(canPos)
	return
}

func (w *World) AtCanonicalPosition(b *Brick) bool {
	canPixelPos := w.PixelPosToCanonicalPixelPos(b.PixelPos)
	return canPixelPos == b.PixelPos
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
	IncludingTop GetObstaclesOption = iota
	ExceptTop
)

func (w *World) GetObstacles(exception *Brick,
	o GetObstaclesOption) (obstacles []Rectangle) {
	obstacles = w.ObstaclesBuffer[:0]
	for j := range w.Bricks {
		otherB := &w.Bricks[j]
		if exception == otherB {
			continue
		}
		// Skip bricks that have the same value.
		if exception.Val == otherB.Val {
			continue
		}

		obstacles = append(obstacles, w.Bricks[j].Bounds)
	}

	bottom := playHeight - w.MarginPixelSize
	top := bottom - w.BrickPixelSize*w.NRows - w.MarginPixelSize*(w.NRows-1)
	left := w.MarginPixelSize
	right := playWidth - w.MarginPixelSize

	bottomRect := Rectangle{Pt{left, bottom}, Pt{right, bottom + 100}}
	topRect := Rectangle{Pt{left, top - 100}, Pt{right, top}}
	leftRect := Rectangle{Pt{left - 100, top}, Pt{left, bottom}}
	rightRect := Rectangle{Pt{right, top}, Pt{right + 100, bottom}}

	obstacles = append(obstacles, bottomRect)
	if o == IncludingTop {
		obstacles = append(obstacles, topRect)
	}
	obstacles = append(obstacles, leftRect)
	obstacles = append(obstacles, rightRect)
	return
}

type MoveType int64

const (
	IgnoreObstacles MoveType = iota
	StopAtFirstObstacleIncludingTop
	StopAtFirstObstacleExceptTop
	SlideOnObstacles
)

// MoveBrick should be the only function that changes the PixelPos of a brick.
// If this convention is followed, it's easy to compute some values for bricks
// only when their position changes, not every time we need them.
func (w *World) MoveBrick(b *Brick, targetPos Pt, nMaxPixels int64, moveType MoveType) (hitObstacle bool) {
	if b.PixelPos == targetPos {
		return false
	}

	if moveType == IgnoreObstacles {
		// Go towards the target pos, without considering any obstacles.
		pts := GetLinePoints(b.PixelPos, targetPos, nMaxPixels)
		ChangePixelPos(b, w, pts[len(pts)-1])
		return false
	}

	if moveType == StopAtFirstObstacleIncludingTop {
		obstacles := w.GetObstacles(b, IncludingTop)
		newR, nPixelsLeft := MoveRect(b.Bounds, targetPos, nMaxPixels, obstacles)
		ChangePixelPos(b, w, newR.Corner1)
		return nPixelsLeft > 0
	}

	if moveType == StopAtFirstObstacleExceptTop {
		obstacles := w.GetObstacles(b, ExceptTop)
		newR, nPixelsLeft := MoveRect(b.Bounds, targetPos, nMaxPixels, obstacles)
		b.PixelPos = newR.Corner1
		ChangePixelPos(b, w, newR.Corner1)
		return nPixelsLeft > 0
	}

	if moveType == SlideOnObstacles {
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

		r := b.Bounds
		obstacles := w.GetObstacles(b, IncludingTop)

		// First, go as far as possible towards the target, in a straight line.
		r, nMaxPixels = MoveRect(b.Bounds, targetPos, nMaxPixels, obstacles)

		// Now, go towards the target's X as much as possible.
		r, nMaxPixels = MoveRect(r, Pt{targetPos.X, r.Corner1.Y},
			nMaxPixels, obstacles)

		// Now, go towards the target's Y as much as possible.
		r, nMaxPixels = MoveRect(r, Pt{r.Corner1.X, targetPos.Y},
			nMaxPixels, obstacles)

		ChangePixelPos(b, w, r.Corner1)
		return true
	}

	panic("unhandled movement type")
}
