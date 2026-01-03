package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
)

// SimulationVersion is the version of the simulation currently implemented
// by the World.
// The simulation is an abstract mapping between input given by the player and
// output received by the player. The input usually consists of mouse positions
// and clicks. The output depends on the current design of the simulation but
// can be things like, the positions of objects (player character, enemies,
// terrain, blocks etc), the status of the objects (health, ammo, state etc),
// the game's status (e.g. score) and whatever else is relevant for the player's
// experience.
// The implementation is not relevant for the SimulationVersion. The
// calculations to go from input to output can change. The exact structs for
// the input and output can change (e.g. int64 changes to int32). As long as
// the information is the same and the mapping is the same, the
// SimulationVersion should remain the same.
// The main purpose of SimulationVersion is to allow for refactoring and
// regression tests. If the SimulationVersion doesn't change, a playthrough
// that was recorded with the same SimulationVersion can be made to be replayed
// with the current simulation code, even if everything else changed.
const SimulationVersion = 99

// World coordinates
// -----------------
//
// The World uses a pixel-based coordinate system where (0, 0) is the top-left
// point. This follows ebitengine's coordinate system on purpose, so that the
// mapping between the World coordinate system and the UI coordinate system is
// as simple as possible. A unit in the game world is expected to be a pixel in
// the UI, which is approximately a pixel on the player's actual device.
//
// The logic is the following:
// - The World defines the size the brick and the space between bricks.
// - The World defines the number of rows and columns.
// - Together, these determine the size of the play area that the UI must
// render.
// - The UI then defines the sizes of its other elements (buttons, menus etc)
// relative to the size of the play area. Together, they define the game area.
// - ebitengine rescales the entire game area to fit the window size on the OS.
//
// This means the size of all elements is decided relative to the size of the
// brick which is decided by the World.
//
// The World expects input in its own coordinate system. Even if the UI uses the
// World coordinates without rescaling, it must account for any margins that it
// adds to the play area that the World is aware of.
//
// Reasoning
// ---------
//
// A game usually has a coordinate system for the World and a coordinate system
// for the user interface. And usually a unit in the game world (e.g. a meter)
// is different from a unit on the screen (a pixel). This is unavoidable in 3D
// games where coordinates in the game world are 3D and the coordinates on the
// screen are 2D.
//
// For a simple 2D game like this, I found it easier to reason about everything
// in pixels. First of all, I want to use integers for coordinates and
// algorithms anyway, to have perfect determinism. Then, the logic is simple
// enough and the world is small enough that all the physics algorithms can be
// pixel based. For example when bricks move I think of them as moving pixel by
// pixel, not in an abstract continuous space.
//
// When you decide the sizes of various visual elements, you have to always
// remember what coordinate system you are in. Also, the absolute sizes are not
// very important since everything gets rescaled by ebitengine in the end. What
// matters is the size of one element relative to the size of another. I found
// it easiest to fix the size of the brick in pixels and build everything on top
// of that.

const NCols = int64(6)
const NRows = int64(8)
const BrickPixelSize = int64(135)
const BrickMarginPixelSize = int64(25)
const PlayAreaWidth = NCols*BrickPixelSize + (NCols-1)*BrickMarginPixelSize
const PlayAreaHeight = NRows*BrickPixelSize + (NRows-1)*BrickMarginPixelSize

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
	Follower
)

type BrickParams struct {
	Pos Pt
	Val int64
}

type ChainParams struct {
	Brick1 int64
	Brick2 int64
}

type Level struct {
	BricksParams          []BrickParams
	ChainsParams          []ChainParams
	TimerDisabled         bool
	AllowOverlappingDrags bool
}

type Brick struct {
	Id  int64
	Val int64
	// This should only be set by SetBrickPos.
	PixelPos     Pt
	State        BrickState
	FallingSpeed int64
	ChainedTo    int64
	// Derived values. These should only ever be read. They are re-computed
	// every time PixelPos changes.
	CanonicalPos      Pt
	CanonicalPixelPos Pt
	Bounds            Rectangle
}

func (w *World) NewBrick(pixelPos Pt, val int64) Brick {
	// Ensure the position is valid.
	Assert(pixelPos.X >= 0)
	Assert(pixelPos.X <= PlayAreaWidth-BrickPixelSize)
	Assert(pixelPos.Y >= 0)
	Assert(pixelPos.Y <= PlayAreaHeight+BrickMarginPixelSize)

	b := Brick{
		Id:    w.NextBrickId,
		Val:   val,
		State: Canonical,
	}
	w.SetBrickPos(&b, pixelPos)
	w.NextBrickId++
	return b
}

func (w *World) SetBrickPos(b *Brick, newPos Pt) {
	if b.ChainedTo > 0 {
		b2 := w.GetBrick(b.ChainedTo)
		if b2.State == Follower {
			dif := newPos.Minus(b.PixelPos)
			w.SetBrickPos(b2, b2.PixelPos.Plus(dif))
		}
	}

	b.PixelPos = newPos
	b.Bounds = BrickBounds(b.PixelPos)
	b.CanonicalPos = PixelPosToCanonicalPos(b.PixelPos)
	b.CanonicalPixelPos = CanonicalPosToPixelPos(b.CanonicalPos)
	// Ensure the new position is valid.
	Assert(b.PixelPos.X >= 0 && b.PixelPos.X < PlayAreaWidth)
	// Ensure the canonical position is valid.
	Assert(b.CanonicalPos.X >= 0 && b.CanonicalPos.X < NCols && b.CanonicalPos.Y >= -1 && b.CanonicalPos.Y <= NRows)
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
	Seed                     int64
	NextBrickId              int64
	DragSpeed                int64
	CanonicalAdjustmentSpeed int64
	BrickFallAcceleration    int64
	Bricks                   []Brick
	DraggingOffset           Pt
	DebugPts                 []Pt
	TimerDisabled            bool
	TimerCooldown            int64
	TimerCooldownIdx         int64
	ComingUpDistanceLeft     int64
	ComingUpSpeed            int64
	ComingUpDeceleration     int64
	State                    WorldState
	PreviousState            WorldState
	SolvedFirstState         bool
	AssertionFailed          bool
	MaxBrickValue            int64
	MaxInitialBrickValue     int64
	ObstaclesBuffer          []Rectangle
	ColumnsBuffer            [][]*Brick
	FirstComingUp            bool
	Score                    int64
	JustMergedBricks         []*Brick
	SlotsBuffer              Mat
	AllowOverlappingDrags    bool
	GroupsBuffer             [31][]*Brick
}

type PlayerInput struct {
	Pos             Pt
	JustPressed     bool
	JustReleased    bool
	TriggerComingUp bool
}

func (p *PlayerInput) EventOccurred() bool {
	return p.JustPressed || p.JustReleased || p.TriggerComingUp
}

// NewWorld creates a world object that is ready for updates.
func NewWorld(seed int64, l Level) (w World) {
	// Set constants and buffers.
	w.NextBrickId = 1
	w.MaxBrickValue = 30
	w.MaxInitialBrickValue = 5
	w.DragSpeed = 100
	w.CanonicalAdjustmentSpeed = 21
	w.BrickFallAcceleration = 2
	w.ComingUpDeceleration = 2
	w.ObstaclesBuffer = make([]Rectangle, NCols*NRows+4)
	w.ColumnsBuffer = make([][]*Brick, NCols)
	for i := range w.ColumnsBuffer {
		w.ColumnsBuffer[i] = make([]*Brick, NRows)
	}
	w.SlotsBuffer = NewMat(Pt{NCols, NRows})
	w.Bricks = make([]Brick, 0, NCols*(NRows+1))
	for i := range w.GroupsBuffer {
		w.GroupsBuffer[i] = make([]*Brick, 0, NCols*(NRows+1))
	}

	// Initialize the world from level parameters.
	w.Seed = seed
	w.RSeed(w.Seed)
	w.TimerDisabled = l.TimerDisabled
	w.AllowOverlappingDrags = l.AllowOverlappingDrags

	w.Bricks = w.Bricks[:0]
	if len(l.BricksParams) == 0 {
		// No bricks specified. Assume we can initialize a regular play.
		w.CreateFirstRowsOfBricks()
		w.ResetTimerCooldown()
		w.TimerCooldownIdx = 0
		w.SolvedFirstState = false
		w.FirstComingUp = true
		w.State = ComingUp
	} else {
		// Bricks specified. Assume this is a test and initialize based on the
		// specifications.
		for i := range l.BricksParams {
			w.Bricks = append(w.Bricks, w.NewBrick(
				l.BricksParams[i].Pos,
				l.BricksParams[i].Val))
		}
		for _, c := range l.ChainsParams {
			ChainBricks(&w.Bricks[c.Brick1], &w.Bricks[c.Brick2])
		}
		w.ResetTimerCooldown()
		w.SolvedFirstState = false
		w.FirstComingUp = false
		w.State = Regular
	}
	return w
}

func (w *World) GetBrick(id int64) *Brick {
	for i := range w.Bricks {
		if w.Bricks[i].Id == id {
			return &w.Bricks[i]
		}
	}
	panic(fmt.Errorf("brick not found: %d", id))
}

func ChainBricks(b1 *Brick, b2 *Brick) {
	Assert(
		(b1.CanonicalPos.Y == b2.CanonicalPos.Y &&
			b1.CanonicalPos.X+1 == b2.CanonicalPos.X) ||
			(b1.CanonicalPos.Y+1 == b2.CanonicalPos.Y &&
				b1.CanonicalPos.X == b2.CanonicalPos.X))
	Assert(b1.ChainedTo == 0)
	Assert(b2.ChainedTo == 0)
	b1.ChainedTo = b2.Id
	b2.ChainedTo = b1.Id
	b2.State = Follower
}

// NewWorldFromPlaythrough checks if the Playthrough has the same simulation
// version as the current code.
func NewWorldFromPlaythrough(p Playthrough) (w World) {
	if p.SimulationVersion != SimulationVersion {
		Check(fmt.Errorf("can't run this playthrough with the current "+
			"simulation - we are at SimulationVersion %d and playthrough "+
			"was generated with SimulationVersion version %d",
			SimulationVersion, p.SimulationVersion))
	}
	w = NewWorld(p.Seed, p.Level)
	return
}

func (w *World) ResetTimerCooldown() {
	// The timer cooldown depends on the maximum brick value currently present
	// on the board. The formula is 11.3 sec + 0.2 sec * maxValue. In terms of
	// frames: 678 + 12 * maxValue
	w.TimerCooldown = 678 + 12*w.CurrentMaxVal()
	w.TimerCooldownIdx = w.TimerCooldown
}

func (w *World) Step(input PlayerInput) {
	w.JustMergedBricks = w.JustMergedBricks[:0]

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
	}
	w.PreviousState = w.State

	// We want to register if the player clicked a brick or released an already
	// dragged brick both during Regular play and during a ComingUp event.
	w.DetermineDraggedBrick(input)

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

func (w *World) DetermineDraggedBrick(input PlayerInput) {
	var dragged *Brick
	for i := range w.Bricks {
		if w.Bricks[i].State == Dragged {
			dragged = &w.Bricks[i]
		}
	}

	if input.JustPressed {
		// Check if there's any brick under the click.
		// Get the closest brick.
		var closest *Brick
		var minDist int64 = math.MaxInt64
		for i := range w.Bricks {
			r := w.Bricks[i].Bounds
			center := r.Min.Plus(r.Max).DivBy(2)
			dist := center.SquaredDistTo(input.Pos)
			if dist < minDist {
				minDist = dist
				closest = &w.Bricks[i]
			}
		}

		// Check if the closest brick is close enough to be dragged.
		minDistForDragging := Sqr(int64(135))
		if minDist <= minDistForDragging {
			// We can check here if dragged == nil. If not, it means that
			// somehow the player clicked a brick, didn't release it and
			// then clicked on another brick. This should not be possible
			// unless there is a hardware failure or the OS/browser/etc
			// doesn't send the game all the hardware signals from the
			// player. I will leave leave the assert here during development
			// to catch errors in my logic. But I will also handle failures
			// gracefully, for when the assert is disabled.
			// Assert(dragged == nil)

			if dragged != nil {
				// Make the previously dragged brick canonical and let the
				// canonical adjustment system handle it.
				dragged.State = Canonical
			}

			if closest.State == Follower {
				dragged = w.GetBrick(closest.ChainedTo)
			} else {
				dragged = closest
			}
			dragged.State = Dragged
			w.DraggingOffset = dragged.Bounds.Min.Minus(input.Pos)
		}
	}

	if input.JustReleased {
		if dragged != nil {
			dragged.State = Canonical
			return
		}
	}
}

func (w *World) StepRegular(justEnteredState bool, input PlayerInput) {
	if !w.TimerDisabled {
		w.TimerCooldownIdx--
		if w.TimerCooldownIdx <= 0 {
			w.State = ComingUp
			return
		}

		if w.NoMoreMergesArePossible() {
			w.TimerCooldownIdx = 0
			w.State = ComingUp
			return
		}
	}

	w.UpdateDraggedBrick(input)
	w.UpdateFallingBricks()
	w.UpdateCanonicalBricks()
	w.MergeBricks()

	// Check if bricks went over the top.
	// This can be possible due to adjustments made in UpdateCanonicalBricks.
	for i := range w.Bricks {
		top := int64(0)
		brickTop := w.Bricks[i].Bounds.Min.Y

		if brickTop < top {
			// The brick is over the top.
			w.State = Lost
			return
		}
	}

	// Disable the check below as we currently do allow intersections to occur
	// in some cases and the strategy is to recover from them. So "solids
	// should never intersect" is no longer a valid invariant to check against.
	//
	// Check if bricks intersect each other or are out of bounds.
	// {
	// 	for i := range w.Bricks {
	// 		obstacles := w.GetObstacles(&w.Bricks[i], IncludingTop)
	// 		brick := BrickBounds(w.Bricks[i].PixelPos)
	// 		// Don't use RectIntersectsRects because I want to be able to
	// 		// put a breakpoint here and see which rect intersects which.
	// 		for j := range obstacles {
	// 			if brick.Intersects(obstacles[j]) {
	// 				Check(fmt.Errorf("solids intersect each other"))
	// 				w.AssertionFailed = true
	// 			}
	// 		}
	// 	}
	// }
}

func (w *World) NoMoreMergesArePossible() bool {
	// Group bricks by their value.
	groups := w.GroupsBuffer
	for i := range groups {
		groups[i] = groups[i][:0]
	}
	for i := range w.Bricks {
		b := &w.Bricks[i]
		groups[b.Val] = append(groups[b.Val], b)
	}

	// Go through each group and check if it contains bricks that could be
	// merged.
	for _, g := range groups {
		for i := range g {
			for j := i + 1; j < len(g); j++ {
				b1 := g[i]
				b2 := g[j]
				// We can merge b1 to b2 if they are not chained to each other.
				if b1.ChainedTo != b2.Id {
					// A merge is possible.
					return false
				}
			}
		}
	}

	return true
}

func (w *World) UpdateDraggedBrick(input PlayerInput) {
	var dragged *Brick
	for i := range w.Bricks {
		if w.Bricks[i].State == Dragged {
			dragged = &w.Bricks[i]
		}
	}

	if dragged == nil {
		return
	}

	if w.AllowOverlappingDrags {
		targetPos := input.Pos.Plus(w.DraggingOffset)
		// Clip targetPos to within the bounds of the game area.
		if targetPos.X < 0 {
			targetPos.X = 0
		}
		if targetPos.Y < 0 {
			targetPos.Y = 0
		}
		if targetPos.X > PlayAreaWidth-BrickPixelSize {
			targetPos.X = PlayAreaWidth - BrickPixelSize
		}
		if targetPos.Y > PlayAreaHeight+BrickMarginPixelSize {
			targetPos.Y = PlayAreaHeight + BrickMarginPixelSize
		}
		w.MoveBrick(dragged, targetPos, w.DragSpeed, IgnoreObstacles)
	} else {
		// Get the set of rectangles the brick must not intersect.
		w.GetObstacles(dragged, IncludingTop, &w.ObstaclesBuffer)

		// If the dragged brick intersects something, it becomes canonical and
		// the behavior of canonical bricks will resolve the intersection.
		if RectIntersectsRects(dragged.Bounds, w.ObstaclesBuffer) {
			dragged.State = Canonical
			return
		}

		// Also check if the follower intersects something.
		if dragged.ChainedTo != 0 {
			b2 := w.GetBrick(dragged.ChainedTo)
			w.GetObstacles(b2, IncludingTop, &w.ObstaclesBuffer)
			if RectIntersectsRects(b2.Bounds, w.ObstaclesBuffer) {
				dragged.State = Canonical
				return
			}
		}

		targetPos := input.Pos.Plus(w.DraggingOffset)
		w.MoveBrick(dragged, targetPos, w.DragSpeed, SlideOnObstacles)
	}
}

func BrickBounds(posPixels Pt) Rectangle {
	return NewRectangle(posPixels,
		posPixels.Plus(Pt{BrickPixelSize, BrickPixelSize}))
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
			b.FallingSpeed, StopAtFirstObstacleExceptTop)
		if hitObstacle {
			// We hit something.
			// The brick becomes canonical.
			b.State = Canonical
			b.FallingSpeed = 0
		}
	}
}

func (w *World) UpdateCanonicalBricks() {
	w.MarkFallingBricks()
	w.ConvergeTowardsCanonicalPositions()
}

// MarkFallingBricks checks if any canonical brick should start falling and
// changes its state.
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

		if b.CanonicalPos.Y == 0 {
			// The brick is already at the bottom, it cannot fall any lower.
			continue
		}

		// Check if the brick or the slot underneath intersects other bricks.
		w.GetObstacles(b, IncludingTop, &w.ObstaclesBuffer)
		r := b.Bounds
		r.Max.Y += BrickPixelSize + BrickMarginPixelSize
		if RectIntersectsRects(r, w.ObstaclesBuffer) {
			continue
		}

		if b.ChainedTo != 0 {
			// Check the chained brick as well.
			// Checking underneath the chained brick is only relevant for a
			// horizontally chained brick but it doesn't hurt logically or
			// computationally to do the same operation for vertically chained
			// bricks as well.
			b2 := w.GetBrick(b.ChainedTo)
			w.GetObstacles(b2, IncludingTop, &w.ObstaclesBuffer)
			r2 := b2.Bounds
			r2.Max.Y += BrickPixelSize + BrickMarginPixelSize
			if RectIntersectsRects(r2, w.ObstaclesBuffer) {
				continue
			}
		}

		// Nothing is sustaining the brick, so it should start falling.
		b.State = Falling
		b.FallingSpeed = 0
	}
}

func (w *World) ConvergeTowardsCanonicalPositions() {
	// Motivation
	// ----------
	//
	// This game requires bricks to stay in their place until they are moved
	// around by the player or they fall because there's nothing underneath
	// them.
	//
	// However, bricks are moved around by the player and often released in
	// non-standard positions. When they fall they might travel a little below
	// the standard position before they hit a brick and need to stop.
	//
	// There is also the issue of edge cases where bricks that should not
	// intersect can be made to intersect by arranging just the right
	// combination of bricks and having merges occur.
	//
	// General solution
	// ----------------
	//
	// The above problems can all be solved by having an algorithm with the
	// following characteristics:
	// - Decide a valid position for each brick.
	// - Minimize travel distances in some way that looks good in the average
	// case.
	// - Have bricks travel each frame towards their positions without
	// considering any obstacles.
	//
	// Design
	// ------
	//
	// The algorithm goes through each brick and assigns it a canonical position
	// that is yet unoccupied. Once the position is occupied, future bricks must
	// be assigned different positions.
	//
	// If two bricks are close to the same canonical positions, we will get a
	// conflict. One must have priority over the other. The other one must be
	// assigned to a different canonical position.
	//
	// The floor and the left and right walls impose hard constraints, bricks
	// cannot move beyond them. So, when there is a conflict, search upwards for
	// the next valid position. Sort the bricks by their Y coordinate, assign
	// positions starting with the lowest Y, going up. This way we hope to
	// achieve minimal movements in the average case.
	//
	// Correctness
	// -----------
	//
	// Since the algorithm always assigns each brick to its own canonical and
	// unoccupied position, it should always push bricks towards a valid state
	// where they don't overlap. Since we go from bottom to top, bricks are not
	// usually assigned positions that have nothing underneath them. If this
	// ever happens momentarily, the brick falls and will then converge to a
	// canonical position that has something underneath it.
	//
	// The biggest concern with the algorithm is for the adjustments to look
	// natural in the average case, and somewhat natural in edge cases. This has
	// been tested manually and I can confirm is looks natural enough.

	// Sort bricks.
	slices.SortStableFunc(w.Bricks, func(b1, b2 Brick) int {
		if b2.PixelPos.Y != b1.PixelPos.Y {
			// Bigger Y has priority (must appear first in the list).
			return cmp.Compare(b2.PixelPos.Y, b1.PixelPos.Y)
		} else {
			// When Y is the same, smaller X has priority (must appear first in
			// the list).
			return cmp.Compare(b1.PixelPos.X, b2.PixelPos.X)
		}
	})

	slots := w.SlotsBuffer
	slots.Reset()

	for i := range w.Bricks {
		b := &w.Bricks[i]
		if b.State != Canonical {
			continue
		}

		// We need to find a position for b, in this column.
		// We start off from b's current position.
		targetCanPos := b.CanonicalPos
		var b2 *Brick
		var targetCanPos2 Pt
		if b.ChainedTo > 0 {
			b2 = w.GetBrick(b.ChainedTo)
			Assert(b2.State == Follower)
			targetCanPos2 = b2.CanonicalPos
		}

		// Find an unoccupied position.
		for {
			if !slots.InBounds(targetCanPos) || !slots.InBounds(targetCanPos2) {
				// The brick is going out of bounds, so just let it go
				// there and trigger game over.
				break
			}

			// Check if the position is already occupied by another brick of a
			// different value.
			otherB := slots.Get(targetCanPos)
			occupied := otherB != nil && otherB.Val != b.Val
			if b2 != nil && !occupied {
				otherB2 := slots.Get(targetCanPos2)
				occupied = otherB2 != nil && otherB2.Val != b2.Val
			}

			if occupied {
				targetCanPos.Y++
				targetCanPos2.Y++
			} else {
				// Found an unoccupied position.
				break
			}
		}

		if slots.InBounds(targetCanPos) {
			slots.Set(targetCanPos, b)
		}
		// If we decided the position of a leader brick, we also decided the
		// position of the follower brick.
		if b2 != nil && slots.InBounds(targetCanPos2) {
			slots.Set(targetCanPos2, b2)
		}

		// Go towards the target pos, without considering any obstacles.
		targetPos := CanonicalPosToPixelPos(targetCanPos)
		w.MoveBrick(b, targetPos, w.CanonicalAdjustmentSpeed,
			IgnoreObstacles)
	}
}

// MergeBricks performs merges between all the bricks that are the same value
// and are close enough.
// Before: some bricks may be close and have the same value.
// After: no bricks are close and have the same value.
func (w *World) MergeBricks() {
	// I don't expect to ever have more than one merge happen in one frame but
	// I feel weird hardcoding that assumption when I can just add a loop to
	// handle that case as well.
	for {
		foundMerge, i, j := w.FindMergingBricks()
		if !foundMerge {
			return
		}
		Assert(i != j)

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
		canPixelPos1 := PixelPosToCanonicalPixelPos(b1.PixelPos)
		dif1 := b1.PixelPos.SquaredDistTo(canPixelPos1)

		canPixelPos2 := PixelPosToCanonicalPixelPos(b2.PixelPos)
		dif2 := b2.PixelPos.SquaredDistTo(canPixelPos2)

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
		w.JustMergedBricks = append(w.JustMergedBricks, brickToUpdate)

		// A merge breaks the chains off the bricks involved in the merge.
		w.UnchainBrick(b1)
		w.UnchainBrick(b2)

		// Update the score.
		w.Score += brickToUpdate.Val

		// Perform the merge.
		brickToUpdate.Val++
		brickToUpdate.State = Canonical
		w.Bricks = Remove(w.Bricks, idxToRemove)
		if brickToUpdate.Val == w.MaxBrickValue {
			w.State = Won
		}
	}
}

func (w *World) UnchainBrick(b *Brick) {
	if b.ChainedTo == 0 {
		return
	}

	b2 := w.GetBrick(b.ChainedTo)
	b.ChainedTo = 0
	b2.ChainedTo = 0

	if b.State == Follower {
		b.State = b2.State
	} else if b2.State == Follower {
		b2.State = b.State
	} else {
		panic("no follower")
	}
}

func (w *World) FindMergingBricks() (foundMerge bool, i, j int) {
	// Two bricks merge if they are close enough for each other.
	// We decide here what "close enough" means.
	mergeDist := Sqr(BrickPixelSize / 3)
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

func (w *World) CreateFirstRowsOfBricks() {
	w.Bricks = w.Bricks[:0]

	// Create the first row.
	for x := range NCols {
		val := w.RInt(1, w.MaxInitialBrickValue-1)
		pos := CanonicalPosToPixelPos(Pt{x, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos, val))
	}

	// Create a row below that will not cause any merges.
	w.CreateNewRowOfBricks(w.MaxInitialBrickValue - 1)

	// Set some brick to have the max value.
	randomIndex := w.RInt(0, int64(len(w.Bricks))-1)
	w.Bricks[randomIndex].Val = w.MaxInitialBrickValue
}

func (w *World) CurrentMaxVal() int64 {
	currentMaxVal := int64(0)
	for i := range w.Bricks {
		if w.Bricks[i].Val > currentMaxVal {
			currentMaxVal = w.Bricks[i].Val
		}
	}
	return currentMaxVal
}

// CreateNewRowOfBricks adds a new row of bricks underneath the existing bricks.
// It will not cause any automatic merges.
// It will generate chains based on the current maxVal.
func (w *World) CreateNewRowOfBricks(maxVal int64) {
	type BrickPair struct {
		b1 *Brick
		b2 *Brick
	}
	var possibleChains []BrickPair
	for x := range NCols {
		// Get a value that is different from the value of the brick right
		// above (if there is a brick right above).
		newPos := CanonicalPosToPixelPos(Pt{x, -1})
		posAbove := Pt{x, 0}
		forbiddenValue := int64(0)
		var brickAbove *Brick
		for i, b := range w.Bricks {
			if b.CanonicalPos == posAbove {
				forbiddenValue = b.Val
				brickAbove = &w.Bricks[i]
			}
		}

		val := int64(0)
		for {
			val = w.RInt(1, maxVal)
			if val != forbiddenValue {
				break
			}
		}

		w.Bricks = append(w.Bricks, w.NewBrick(newPos, val))

		if brickAbove != nil &&
			brickAbove.State == Canonical &&
			brickAbove.ChainedTo == 0 {
			// Vertical chain possible.
			possibleChains = append(possibleChains,
				BrickPair{
					&w.Bricks[len(w.Bricks)-1],
					brickAbove})
		}
		if x > 0 {
			// Horizontal chain possible.
			possibleChains = append(possibleChains,
				BrickPair{
					&w.Bricks[len(w.Bricks)-2],
					&w.Bricks[len(w.Bricks)-1]})
		}
	}

	// Add chains, if we are at 10 or more.
	currentMaxVal := w.CurrentMaxVal()
	if currentMaxVal < 10 {
		return
	}

	// Compute how many chains there will be in this new row.
	// On average, the number of chains per row = (currentMaxVal - 1) / 10.
	// Since this will usually be a decimal number, we add the integral part
	// and then we increase by 1 randomly, where the chance of increasing by 1
	// depends on the decimal part.
	nChainsToAdd := (currentMaxVal - 1) / 10
	if w.RInt(0, 10) < (currentMaxVal-1)%10 {
		nChainsToAdd++
	}

	// Add chains randomly to valid chain positions until we run out of chains
	// to add or we run out of valid chain positions.
	for {
		if nChainsToAdd == 0 {
			break
		}
		if len(possibleChains) == 0 {
			// This can happen in rare cases. The maximum value before the game
			// is won is 29, so the maximum value for nChainsToAdd is 3.
			// If the new row is [1, 2, 3, 4, 5, 6] you can place one chain
			// between bricks 2 and 3, then another between 4 and 5, and no more
			// horizontal chains are possible. If you can't add vertical chains
			// betewen 1 or 6 and the bricks above them, you just ran out of
			// places to add chains to.
			break
		}
		chosenOne := w.RInt(int64(0), int64(len(possibleChains)-1))
		c := possibleChains[chosenOne]
		ChainBricks(c.b1, c.b2)
		possibleChains = Remove(possibleChains, int(chosenOne))
		nChainsToAdd--

		// A brick might appear multiple times in possibleChains, because it
		// may be chained to the brick above or the brick next to it. So remove
		// entries in possibleChains which include the recently chained bricks.
		n := 0
		for i, p := range possibleChains {
			containsNewlyChainedBricks :=
				p.b1 == c.b1 || p.b1 == c.b2 || p.b2 == c.b1 || p.b2 == c.b2
			if !containsNewlyChainedBricks {
				possibleChains[n] = possibleChains[i]
				n++
			}
		}
		possibleChains = possibleChains[:n]
	}

	if w.NoMoreMergesArePossible() {
		// We just added a new row, but no merges are possible, which means
		// a new row will appear right after this one. That can be
		// surprising and unfair to the player.
		// Create a possible merge by making the last brick able to merge with
		// some other brick.
		lastB := &w.Bricks[len(w.Bricks)-1]

		// Make sure the last brick is unchained.
		if lastB.ChainedTo != 0 {
			w.UnchainBrick(lastB)
		}

		// Set it to the first value that is different from the value of the
		// brick above.
		posAbove := lastB.CanonicalPos
		posAbove.Y++
		for _, b := range w.Bricks {
			// There must be at least one brick with a value different from
			// the forbidden value, we have 5 other bricks in the new row,
			// they can't all have the forbiddenValue but also not allow
			// any merges.
			if b.CanonicalPos != posAbove {
				lastB.Val = b.Val
				break
			}
		}
	}

	// Check that we solved the issue.
	Assert(!w.NoMoreMergesArePossible())
}

// StepComingUp creates a new row of bricks under the existing bricks and moves
// all existing bricks upwards, for a short time. If a brick goes over the top
// the game ends. This is the main game-over condition.
func (w *World) StepComingUp(justEnteredState bool) {
	if justEnteredState {
		// We have to compute the speed we need to start with in order to
		// decelerate by the desired deceleration rate and travel the desired
		// distance in the desired time and reach the destination with speed
		// zero or close to zero.
		// In order to do this, reverse the problem: if we start with speed 0
		// and keep increasing the speed, what speed to we reach by the time we
		// cover the distance?
		totalDist := BrickPixelSize + BrickMarginPixelSize
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
		w.ComingUpDistanceLeft = BrickPixelSize + BrickMarginPixelSize
		if w.FirstComingUp {
			w.FirstComingUp = false
		} else {
			w.CreateNewRowOfBricks(w.CurrentMaxVal() - 2)
		}

		w.ResetTimerCooldown()
	}

	// In the last step, the speed might be higher than the distance left.
	// In this case, just travel the exact distance left.
	if w.ComingUpSpeed > w.ComingUpDistanceLeft {
		w.ComingUpSpeed = w.ComingUpDistanceLeft
	}
	for i := range w.Bricks {
		newPos := w.Bricks[i].PixelPos
		newPos.Y -= w.ComingUpSpeed
		if w.Bricks[i].State == Follower {
			// Skip follower bricks otherwise they get moved twice.
			continue
		}
		w.SetBrickPos(&w.Bricks[i], newPos)
	}
	w.ComingUpDistanceLeft -= w.ComingUpSpeed
	w.ComingUpSpeed -= w.ComingUpDeceleration

	// Check if we're done.
	if w.ComingUpDistanceLeft == 0 {
		// Check if bricks went over the top.
		for i := range w.Bricks {
			b := &w.Bricks[i]
			top := int64(0)
			brickTop := w.Bricks[i].Bounds.Min.Y

			if brickTop >= top {
				// The brick is not over the top.
				continue
			}

			// The brick is over the top. Try to move it down so that it's not
			// over the top anymore. This should generally work only for
			// dragged bricks and bricks who were recently dragged and now are
			// in the middle of adjusting their position, because only those
			// bricks will have space to be moved down.
			if b.State == Follower {
				b = w.GetBrick(b.ChainedTo)
			}
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

func PixelPosToCanonicalPos(pixelPos Pt) (canPos Pt) {
	l := float64(BrickPixelSize + BrickMarginPixelSize)
	canPos.X = int64(math.Round(float64(pixelPos.X) / l))
	canPos.Y = int64(math.Round(
		float64(PlayAreaHeight-pixelPos.Y+BrickMarginPixelSize)/l - 1))
	return
}

func CanonicalPosToPixelPos(canPos Pt) (pixelPos Pt) {
	l := BrickPixelSize + BrickMarginPixelSize
	pixelPos.X = canPos.X * l
	pixelPos.Y = PlayAreaHeight - (canPos.Y+1)*l + BrickMarginPixelSize
	return
}

func PixelPosToCanonicalPixelPos(pixelPos Pt) (canPixelPos Pt) {
	canPos := PixelPosToCanonicalPos(pixelPos)
	canPixelPos = CanonicalPosToPixelPos(canPos)
	return
}

type GetObstaclesOption int64

const (
	IncludingTop GetObstaclesOption = iota
	ExceptTop
)

// GetObstacles returns all the obstacles for a certain brick, as rectangles.
// This includes walls and other bricks that have different values than b.
func (w *World) GetObstacles(
	b *Brick,
	o GetObstaclesOption,
	buffer *[]Rectangle) {

	obstacles := (*buffer)[:0]
	for j := range w.Bricks {
		otherB := &w.Bricks[j]
		if otherB == b || otherB.Id == b.ChainedTo {
			continue
		}
		// Skip bricks that have the same value.
		if b.Val == otherB.Val {
			continue
		}

		obstacles = append(obstacles, w.Bricks[j].Bounds)
	}

	bottom := PlayAreaHeight
	top := bottom - PlayAreaHeight
	left := int64(0)
	right := PlayAreaWidth

	bottomRect := NewRectangle(Pt{left, bottom}, Pt{right, bottom + 100})
	topRect := NewRectangle(Pt{left, top - 100}, Pt{right, top})
	leftRect := NewRectangle(Pt{left - 100, top}, Pt{left, bottom})
	rightRect := NewRectangle(Pt{right, top}, Pt{right + 100, bottom})

	obstacles = append(obstacles, bottomRect)
	if o == IncludingTop {
		obstacles = append(obstacles, topRect)
	}
	obstacles = append(obstacles, leftRect)
	obstacles = append(obstacles, rightRect)
	*buffer = obstacles
}

type MoveType int64

const (
	IgnoreObstacles MoveType = iota
	StopAtFirstObstacleExceptTop
	SlideOnObstacles
)

// MoveBrick should be the only function that changes the position of a brick.
func (w *World) MoveBrick(b *Brick, targetPos Pt, nMaxPixels int64,
	moveType MoveType) (hitObstacle bool) {
	// I assume that only non-follower bricks will ever be moved.
	Assert(b.State != Follower)

	if b.PixelPos == targetPos {
		return false
	}

	if moveType == IgnoreObstacles {
		// Go towards the target pos, without considering any obstacles.
		pts := GetLinePoints(b.PixelPos, targetPos, nMaxPixels)
		w.SetBrickPos(b, pts[len(pts)-1])
		return false
	}

	if moveType == StopAtFirstObstacleExceptTop {
		nPixelsLeft := w.MoveBrickHelper(b, targetPos, nMaxPixels,
			ExceptTop)
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
		// 2. Move the brick in small steps and check if it collides with
		// anything after each move. (iterative solution)
		//
		// I will go for the iterative solution for now, because it's more
		// straightforward for me to come up with it. I'm simulating a process
		// that I imagine in an iterative way (the brick "moves towards the
		// target").
		//
		// For the iterative solution, you can do it with floats or integers.
		// I will do it with integers and move the brick pixel by pixel. I do
		// this because I don't want to use floats in the world logic and I may
		// be able to afford the computational cost.
		//
		// Only travel a limited number of pixels, to have the effect of a
		// brick's "travel speed". The travel speed is not that noticeable when
		// moving the brick around in empty space. But if the brick was
		// previously blocked by something on its right and the user lifts it up
		// to the point where now it can go a long way through empty space to
		// reach the mouse position, it is very visible if the brick "travels"
		// or "teleports" and the effect of the brick travelling is more
		// pleasant. It gives more of a feeling that it is an actual solid
		// object in solid space on which forces are acting.

		// First, go as far as possible towards the target, in a straight line.
		nPixelsLeft := w.MoveBrickHelper(b, targetPos, nMaxPixels, IncludingTop)

		// Now, go towards the target's X as much as possible.
		targetPosX := Pt{targetPos.X, b.Bounds.Min.Y}
		nPixelsLeft = w.MoveBrickHelper(b, targetPosX, nPixelsLeft, IncludingTop)

		// Now, go towards the target's Y as much as possible.
		targetPosY := Pt{b.Bounds.Min.X, targetPos.Y}
		nPixelsLeft = w.MoveBrickHelper(b, targetPosY, nPixelsLeft, IncludingTop)

		return nPixelsLeft > 0
	}

	panic("unhandled movement type")
}

// MoveBrickHelper moves a brick while considering obstacles, and takes into
// account the follower brick if we are dealing with chained bricks.
func (w *World) MoveBrickHelper(
	b *Brick,
	targetPos Pt,
	nMaxPixels int64,
	o GetObstaclesOption,
) (nPixelsLeft int64) {

	// Check how much the leader brick can move.
	w.GetObstacles(b, o, &w.ObstaclesBuffer)
	newR, nPixelsLeft := MoveRect(b.Bounds, targetPos, nMaxPixels,
		w.ObstaclesBuffer)
	dif := newR.Min.Minus(b.Bounds.Min)

	if b.ChainedTo > 0 {
		// Get the follower brick.
		b2 := w.GetBrick(b.ChainedTo)
		targetPos2 := b2.PixelPos.Plus(targetPos.Minus(b.PixelPos))

		// Check how much the follower brick can move.
		w.GetObstacles(b2, o, &w.ObstaclesBuffer)
		newR2, nPixelsLeft2 := MoveRect(b2.Bounds, targetPos2, nMaxPixels,
			w.ObstaclesBuffer)
		dif2 := newR2.Min.Minus(b2.Bounds.Min)

		// If the follower brick can move less than the leader brick, limit the
		// movement.
		if nPixelsLeft2 > nPixelsLeft {
			nPixelsLeft = nPixelsLeft2
			dif = dif2
		}
	}

	// Actually change the brick's position.
	w.SetBrickPos(b, b.PixelPos.Plus(dif))
	return
}
