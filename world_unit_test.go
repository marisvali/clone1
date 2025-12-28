package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"slices"
	"testing"
)

// RPos is a utility for getting a valid random position for a brick.
func RPos() (p Pt) {
	p.X = RInt(0, PlayAreaWidth-BrickPixelSize)
	p.Y = RInt(0, PlayAreaHeight-BrickPixelSize)
	return
}

// RPosLeader is a utility for getting a valid random position for a brick that
// is meant to be the leader brick of a chained brick.
func RPosLeader() (p Pt) {
	p.X = RInt(0, PlayAreaWidth-BrickPixelSize-BrickPixelSize-BrickMarginPixelSize)
	p.Y = RInt(BrickPixelSize+BrickMarginPixelSize, PlayAreaHeight-BrickPixelSize)
	return
}

func TestNewBrick(t *testing.T) {
	var w World

	// Is the brick created correctly?
	b := w.NewBrick(Pt{200, 327}, 3)
	assert.Equal(t, Pt{200, 327}, b.PixelPos)
	assert.Equal(t, Pt{1, 5}, b.CanonicalPos)
	assert.Equal(t, int64(3), b.Val)

	// If I create many bricks, will all the ids be unique?
	RSeed(0)
	var bricks []Brick
	for range 100000 {
		b = w.NewBrick(Pt{RInt(0, PlayAreaWidth-BrickPixelSize),
			RInt(0, PlayAreaHeight-BrickPixelSize)}, RInt(1, 30))
		bricks = append(bricks, b)
	}

	idExists := map[int64]bool{}
	for _, b = range bricks {
		assert.False(t, idExists[b.Id])
		idExists[b.Id] = true
	}
}

func TestSetPixelPos(t *testing.T) {
	RSeed(0)

	// Does setting the position set all the variables I expect of a brick?
	// Ridiculous check, but check.
	for range 10 {
		var w World
		var b Brick
		pixelPos := RPos()
		canonicalPos := PixelPosToCanonicalPos(pixelPos)
		canonicalPixelPos := CanonicalPosToPixelPos(canonicalPos)
		bounds := Rectangle{
			Min: pixelPos,
			Max: pixelPos.Plus(Pt{BrickPixelSize, BrickPixelSize})}

		w.SetBrickPos(&b, pixelPos)
		assert.Equal(t, pixelPos, b.PixelPos)
		assert.Equal(t, canonicalPos, b.CanonicalPos)
		assert.Equal(t, canonicalPixelPos, b.CanonicalPixelPos)
		assert.Equal(t, bounds, b.Bounds)
	}

	// Does setting the position of the brick also set the position of the
	// chained brick?
	for range 100 {
		var w World
		w.Bricks = make([]Brick, 2)
		b1 := &w.Bricks[0]
		b2 := &w.Bricks[1]

		// Set ids.
		b1.Id = RInt(0, 10000)
		b2.Id = b1.Id + RInt(1, 10000)
		state1 := BrickState(RInt(0, 2))
		b1.State = state1
		b2.State = BrickState(RInt(0, 2))

		// Set positions. The bricks are unchained still so we're using the
		// SetBrickPos functionality verified above.
		w.SetBrickPos(b1, RPosLeader())
		b2Pos := b1.PixelPos
		if RInt(0, 1) == 0 {
			b2Pos.X += BrickPixelSize + BrickMarginPixelSize
		} else {
			b2Pos.Y -= BrickPixelSize + BrickMarginPixelSize
		}
		w.SetBrickPos(b2, b2Pos)

		// Chain bricks.
		ChainBricks(b1, b2)

		// Check if now that bricks are chained, changing one brick changes
		// the other.
		oldPos1 := b1.PixelPos
		newPos1 := RPosLeader()
		oldPos2 := b2.PixelPos
		dif := newPos1.Minus(oldPos1)
		newPos2 := oldPos2.Plus(dif)

		w.SetBrickPos(b1, newPos1)
		assert.Equal(t, newPos2, b2.PixelPos)
	}

	assert.Equal(t, true, true)
}

func TestEventOccurred(t *testing.T) {
	// Most ridiculous test yet.
	var p PlayerInput
	assert.False(t, p.EventOccurred())
	p.JustPressed = true
	assert.True(t, p.EventOccurred())
	p = PlayerInput{}
	p.JustReleased = true
	assert.True(t, p.EventOccurred())
	p = PlayerInput{}
	p.TriggerComingUp = true
	assert.True(t, p.EventOccurred())
	p = PlayerInput{}
	p.JustPressed = true
	p.JustReleased = true
	p.TriggerComingUp = true
	assert.True(t, p.EventOccurred())
}

func TestNewWorld(t *testing.T) {
	for range 10 {
		w := NewWorld(RInt(0, 10000), Level{})
		require.Equal(t, 12, len(w.Bricks))
		for range 3000 {
			require.NotPanics(t, func() {
				w.Step(PlayerInput{})
			})
		}
	}
}

func TestGetBrick(t *testing.T) {
	var w World
	assert.Panics(t, func() { w.GetBrick(int64(12)) })

	w.Bricks = make([]Brick, 1)
	w.Bricks[0].Id = 3
	assert.Panics(t, func() { w.GetBrick(int64(12)) })
	assert.Equal(t, &w.Bricks[0], w.GetBrick(int64(3)))

	w.Bricks = make([]Brick, 4)
	w.Bricks[0].Id = 3
	w.Bricks[1].Id = 13
	w.Bricks[2].Id = 1
	w.Bricks[3].Id = 25
	assert.Equal(t, &w.Bricks[0], w.GetBrick(int64(3)))
	assert.Equal(t, &w.Bricks[1], w.GetBrick(int64(13)))
	assert.Equal(t, &w.Bricks[3], w.GetBrick(int64(25)))
}

func TestChainBricks(t *testing.T) {
	RSeed(0)

	// Do bricks really get chained (reference each other)?
	// Does the second brick really become a follower?
	for range 10 {
		var b1, b2 Brick
		b2.CanonicalPos.X = 1
		b1.Id = RInt(0, 10000)
		b2.Id = b1.Id + RInt(1, 10000)
		state1 := BrickState(RInt(0, 2))
		b1.State = state1
		b2.State = BrickState(RInt(0, 2))
		ChainBricks(&b1, &b2)
		assert.Equal(t, b1.ChainedTo, b2.Id)
		assert.Equal(t, b2.ChainedTo, b1.Id)
		assert.Equal(t, b1.State, state1)
		assert.Equal(t, b2.State, Follower)
	}
}

func TestNewWorldFromPlaythrough(t *testing.T) {
	p := Playthrough{}
	p.SimulationVersion = 25
	require.Panics(t, func() {
		NewWorldFromPlaythrough(p)
	})
	p.SimulationVersion = SimulationVersion
	require.NotPanics(t, func() {
		NewWorldFromPlaythrough(p)
	})
}

func TestResetTimerCooldown(t *testing.T) {
	// Is the index set?
	var w World
	w.TimerCooldownIdx = 432
	w.ResetTimerCooldown()
	assert.Equal(t, w.TimerCooldown, w.TimerCooldownIdx)

	// Does the cooldown increase as max value increases?
	w.Bricks = make([]Brick, 1)
	prevCooldown := int64(0)
	for i := int64(5); i < 30; i++ {
		w.Bricks[0].Val = i
		w.ResetTimerCooldown()
		assert.Greater(t, w.TimerCooldown, prevCooldown)
		prevCooldown = w.TimerCooldown
	}
}

func TestStep(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestDetermineDraggedBrick(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestStepRegular(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestNoMoreMergesArePossible(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestUpdateDraggedBrick(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestBrickBounds(t *testing.T) {
	// Ridiculous test. The value is that it made me double check this trivial
	// function again.
	for range 10 {
		pos := RPos()
		b := BrickBounds(pos)
		assert.Equal(t, pos, b.Min)
		assert.Equal(t, BrickPixelSize, b.Width())
		assert.Equal(t, BrickPixelSize, b.Height())
	}
}

func TestUpdateFallingBricks(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestUpdateCanonicalBricks(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestMarkFallingBricks(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestConvergeTowardsCanonicalPositions(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestMergeBricks(t *testing.T) {
	RSeed(0)
	{
		// Find mergeable bricks among many.
		for range 100 {
			// Fill the world with bricks.
			var w World
			w.NextBrickId = 1
			for i := 1; i < 30; i++ {
				b := w.NewBrick(RPos(), int64(i))
				b.State = BrickState(RInt(0, 2))
				w.Bricks = append(w.Bricks, b)
			}

			// Chain some of the bricks.
			for range RInt(0, 7) {
				// Choose one to be the leader.
				var b1 *Brick
				for {
					b1 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if b1.ChainedTo == 0 {
						break
					}
				}
				w.SetBrickPos(b1, RPosLeader())

				// Choose one that will be the follower.
				var b2 *Brick
				for {
					b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if b2.ChainedTo == 0 && b1 != b2 {
						break
					}
				}
				// Set the position of the follower to be suitable for chaining.
				b2Pos := b1.PixelPos
				if RInt(0, 1) == 0 {
					b2Pos.X += BrickPixelSize + BrickMarginPixelSize
				} else {
					b2Pos.Y -= BrickPixelSize + BrickMarginPixelSize
				}
				w.SetBrickPos(b2, b2Pos)

				// Chain bricks.
				ChainBricks(b1, b2)
			}

			alreadyMergeable := []*Brick{}
			for range RInt(1, 3) {
				// Choose two different bricks.
				var b1 *Brick
				for {
					b1 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if !slices.Contains(alreadyMergeable, b1) {
						break
					}
				}
				// Choose one that is different and is not a follower.
				var b2 *Brick
				for {
					b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if !slices.Contains(alreadyMergeable, b2) &&
						b2.State != Follower && b1 != b2 &&
						b2.Id != b1.ChainedTo {
						break
					}
				}

				// Put b1 at a position that is safe enough to put a chained brick
				// close to it.
				w.SetBrickPos(b1, Pt{RInt(100, PlayAreaWidth-320),
					RInt(200, PlayAreaHeight-200)})

				// Put b2 close to b1.
				pos1 := b1.PixelPos
				pos2 := pos1.Plus(Pt{RInt(-25, 25), RInt(-25, 25)})
				w.SetBrickPos(b2, pos2)

				// Make b2 have the same value as b1.
				b2.Val = b1.Val

				alreadyMergeable = append(alreadyMergeable, b1)
				alreadyMergeable = append(alreadyMergeable, b2)
			}

			foundMerge, _, _ := w.FindMergingBricks()
			assert.True(t, foundMerge)

			w.MergeBricks()

			foundMerge, _, _ = w.FindMergingBricks()
			assert.False(t, foundMerge)
		}
	}
}

func TestUnchainBrick(t *testing.T) {
	RSeed(0)

	for range 1 {
		// Fill the world with bricks.
		var w World
		for range 100 {
			b := w.NewBrick(RPos(), RInt(1, 30))
			b.State = BrickState(RInt(0, 2))
			w.Bricks = append(w.Bricks, b)
		}

		// Choose one to be the leader.
		b1 := &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
		// Choose one that will be the follower.
		var b2 *Brick
		for {
			b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
			if b1 != b2 {
				break
			}
		}
		// Set the position of the follower to be suitable for chaining.
		b2Pos := b1.PixelPos
		if RInt(0, 1) == 0 {
			b2Pos.X += BrickPixelSize + BrickMarginPixelSize
		} else {
			b2Pos.Y -= BrickPixelSize + BrickMarginPixelSize
		}
		w.SetBrickPos(b2, b2Pos)

		// Chain bricks.
		ChainBricks(b1, b2)

		// Unchain bricks.
		w.UnchainBrick(b1)

		// Check that the bricks are properly unchained.
		// No more references.
		assert.Equal(t, b1.ChainedTo, int64(0))
		assert.Equal(t, b2.ChainedTo, int64(0))
		// And the follower is no longer a follower.
		assert.Equal(t, b1.State, b2.State)
	}
}

func TestFindMergingBricks(t *testing.T) {
	RSeed(0)
	{
		// Insufficient bricks = no merge.
		var w World
		foundMerge, _, _ := w.FindMergingBricks()
		assert.Equal(t, false, foundMerge)
	}
	{
		// Insufficient bricks = no merge.
		var w World
		w.Bricks = append(w.Bricks, w.NewBrick(RPos(), 3))
		foundMerge, _, _ := w.FindMergingBricks()
		assert.Equal(t, false, foundMerge)
	}
	{
		// Random positions, different values = no merge.
		for range 10 {
			var w World
			w.Bricks = append(w.Bricks, w.NewBrick(RPos(), 3))
			w.Bricks = append(w.Bricks, w.NewBrick(RPos(), 4))
			foundMerge, _, _ := w.FindMergingBricks()
			assert.Equal(t, false, foundMerge)
		}
	}
	{
		// Same position, different values = no merge.
		var w World
		pos := RPos()
		w.Bricks = append(w.Bricks, w.NewBrick(pos, 3))
		w.Bricks = append(w.Bricks, w.NewBrick(pos, 4))
		foundMerge, _, _ := w.FindMergingBricks()
		assert.Equal(t, false, foundMerge)
	}
	{
		// Same position, same value = merge.
		for range 10 {
			var w World
			pos := RPos()
			w.Bricks = append(w.Bricks, w.NewBrick(pos, 3))
			w.Bricks = append(w.Bricks, w.NewBrick(pos, 3))
			foundMerge, i, j := w.FindMergingBricks()
			assert.Equal(t, true, foundMerge)
			assert.True(t, i == 0 && j == 1 || i == 1 && j == 0)
		}
	}
	{
		// Close positions, same value = merge.
		for range 100 {
			var w World
			pos1 := Pt{RInt(100, PlayAreaWidth-320),
				RInt(200, PlayAreaHeight-200)}
			pos2 := pos1.Plus(Pt{RInt(-25, 25), RInt(-25, 25)})
			val := RInt(1, 30)
			w.Bricks = append(w.Bricks, w.NewBrick(pos1, val))
			w.Bricks = append(w.Bricks, w.NewBrick(pos2, val))
			foundMerge, i, j := w.FindMergingBricks()
			assert.Equal(t, true, foundMerge)
			assert.True(t, i == 0 && j == 1 || i == 1 && j == 0)
		}
	}
	{
		// Find mergeable bricks among many.
		for range 100 {
			// Fill the world with bricks.
			var w World
			w.NextBrickId = 1
			for i := 1; i < 30; i++ {
				b := w.NewBrick(RPos(), int64(i))
				b.State = BrickState(RInt(0, 2))
				w.Bricks = append(w.Bricks, b)
			}
			// Chain some of the bricks.
			for range RInt(0, 7) {
				// Choose one to be the leader.
				var b1 *Brick
				for {
					b1 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if b1.ChainedTo == 0 {
						break
					}
				}
				w.SetBrickPos(b1, RPosLeader())

				// Choose one that will be the follower.
				var b2 *Brick
				for {
					b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
					if b2.ChainedTo == 0 && b1 != b2 {
						break
					}
				}
				// Set the position of the follower to be suitable for chaining.
				b2Pos := b1.PixelPos
				if RInt(0, 1) == 0 {
					b2Pos.X += BrickPixelSize + BrickMarginPixelSize
				} else {
					b2Pos.Y -= BrickPixelSize + BrickMarginPixelSize
				}
				w.SetBrickPos(b2, b2Pos)

				// Chain bricks.
				ChainBricks(b1, b2)
			}

			// Choose two different bricks.
			b1 := &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
			// Choose one that is different and is not a follower.
			var b2 *Brick
			for {
				b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
				if b2.State != Follower && b1 != b2 {
					break
				}
			}

			// Put b1 at a position that is safe enough to put a chained brick
			// close to it.
			w.SetBrickPos(b1, Pt{RInt(100, PlayAreaWidth-320),
				RInt(200, PlayAreaHeight-200)})

			// Put b2 close to b1.
			pos1 := b1.PixelPos
			pos2 := pos1.Plus(Pt{RInt(-25, 25), RInt(-25, 25)})
			w.SetBrickPos(b2, pos2)

			// Make b2 have the same value as b1.
			b2.Val = b1.Val

			// Get the indexes of the two bricks.
			var b1Idx, b2Idx int
			for i := range w.Bricks {
				if b1.Id == w.Bricks[i].Id {
					b1Idx = i
				}
				if b2.Id == w.Bricks[i].Id {
					b2Idx = i
				}
			}

			foundMerge, i, j := w.FindMergingBricks()
			assert.Equal(t, true, foundMerge)
			assert.True(t, i == b1Idx && j == b2Idx || i == b2Idx && j == b1Idx)
		}
	}
}

func TestCreateFirstRowsOfBricks(t *testing.T) {
	RSeed(0)
	for range 100 {
		w := NewWorld(0, Level{})
		w.Bricks = w.Bricks[:0]

		w.CreateFirstRowsOfBricks()

		// Check that we have 2 rows in the beginning.
		require.Equal(t, 12, len(w.Bricks))

		// Check that the max val is 5.
		require.Equal(t, int64(5), w.CurrentMaxVal())

		// Check that no merges are possible.
		w.TimerDisabled = true
		for range 200 {
			w.Step(PlayerInput{})
			found, _, _ := w.FindMergingBricks()
			require.False(t, found)
		}
	}
}

func TestCurrentMaxVal(t *testing.T) {
	var w World
	w.Bricks = make([]Brick, 0)
	assert.Equal(t, int64(0), w.CurrentMaxVal())

	w.Bricks = make([]Brick, 1)
	w.Bricks[0].Val = 17
	assert.Equal(t, int64(17), w.CurrentMaxVal())

	w.Bricks = make([]Brick, 3)
	w.Bricks[0].Val = 17
	w.Bricks[1].Val = 23
	w.Bricks[2].Val = 12
	assert.Equal(t, int64(23), w.CurrentMaxVal())

	w.Bricks = make([]Brick, 3)
	w.Bricks[0].Val = 17
	w.Bricks[1].Val = 23
	w.Bricks[2].Val = 30
	assert.Equal(t, int64(30), w.CurrentMaxVal())
}

func TestCreateNewRowOfBricks(t *testing.T) {
	RSeed(0)

	for range 100 {
		l := Level{}
		for y := range RInt(1, 4) {
			for x := range int64(6) {
				if RInt(0, 10) == 0 {
					continue
				}
				var bp BrickParams
				bp.Val = y*6 + x + 1
				bp.Pos = CanonicalPosToPixelPos(Pt{x, y})
				l.BricksParams = append(l.BricksParams, bp)
			}
		}
		w := NewWorld(RInt(0, 10000), l)
		found1, _, _ := w.FindMergingBricks()
		require.False(t, found1)

		prevLen := len(w.Bricks)
		maxVal := int64(20)
		w.CreateNewRowOfBricks(maxVal)

		// Check the correct number of new bricks was added.
		require.Equal(t, prevLen+6, len(w.Bricks))

		// Check that their value is less than maxVal.
		for i := prevLen; i < len(w.Bricks); i++ {
			require.GreaterOrEqual(t, maxVal, w.Bricks[i].Val)
			require.Equal(t, int64(-1), w.Bricks[i].CanonicalPos.Y)
		}

		// Count chains.
		nChains := 0
		for i := prevLen; i < len(w.Bricks); i++ {
			if w.Bricks[i].ChainedTo != 0 {
				nChains++
			}
		}
		currentMaxVal := w.CurrentMaxVal()
		if currentMaxVal < 10 {
			require.Equal(t, nChains, 0)
		} else if currentMaxVal < 21 {
			require.GreaterOrEqual(t, nChains, 1)
		} else {
			require.GreaterOrEqual(t, nChains, 2)
		}

		// Check that no new merges are possible.
		w.TimerDisabled = true
		for range 200 {
			w.Step(PlayerInput{})
			found, _, _ := w.FindMergingBricks()
			require.False(t, found)
		}
	}
}

func TestStepComingUp(t *testing.T) {
	// Check that it creates a new row of bricks.
	{
		w := NewWorld(0, Level{})
		for range 300 {
			// Let the first bricks finish moving up.
			w.Step(PlayerInput{})
		}
		prevLen := len(w.Bricks)
		w.State = ComingUp
		w.StepComingUp(true)
		require.Equal(t, prevLen+6, len(w.Bricks))
	}

	// Check that it moves bricks up.
	{
		w := NewWorld(0, Level{})
		for range 300 {
			// Let the first bricks finish moving up.
			w.Step(PlayerInput{})
		}
		w.State = ComingUp
		w.StepComingUp(true)

		var prevPositions []Pt
		for _, b := range w.Bricks {
			prevPositions = append(prevPositions, b.PixelPos)
		}
		for range 1000 {
			w.StepComingUp(false)
			var currentPositions []Pt
			for _, b := range w.Bricks {
				currentPositions = append(currentPositions, b.PixelPos)
			}
			for i := range currentPositions {
				require.Equal(t, currentPositions[i].X, prevPositions[i].X)
				require.Less(t, currentPositions[i].Y, prevPositions[i].Y)
			}

			// Check that it ends the movement and switches to Regular state.
			if w.State == Regular {
				break
			}
		}
		require.Equal(t, Regular, w.State)
	}
}

func TestPixelPosToCanonicalPos(t *testing.T) {
	RSeed(0)

	// Does it truly give me the closest canonical pos?
	for range 1000 {
		pixelPos := RPos()
		minDist := int64(math.MaxInt64)
		var closestCanPos []Pt
		for x := int64(0); x < 6; x++ {
			for y := int64(0); y < 8; y++ {
				canPos := Pt{x, y}
				canPixelPos := CanonicalPosToPixelPos(canPos)
				dist := pixelPos.SquaredDistTo(canPixelPos)
				if dist == minDist {
					closestCanPos = append(closestCanPos, canPos)
				}
				if dist < minDist {
					closestCanPos = closestCanPos[:0]
					minDist = dist
					closestCanPos = append(closestCanPos, canPos)
				}
			}
		}

		assert.Contains(t, closestCanPos, PixelPosToCanonicalPos(pixelPos))
	}
}

func TestCanonicalPosToPixelPos(t *testing.T) {
	// Does the mapping work for a few points?
	assert.Equal(t, Pt{0, PlayAreaHeight - BrickPixelSize}, CanonicalPosToPixelPos(Pt{0, 0}))
	assert.Equal(t, Pt{0, PlayAreaHeight - BrickPixelSize - BrickPixelSize - BrickMarginPixelSize}, CanonicalPosToPixelPos(Pt{0, 1}))
	assert.Equal(t, Pt{BrickPixelSize + BrickMarginPixelSize, PlayAreaHeight - BrickPixelSize}, CanonicalPosToPixelPos(Pt{1, 0}))
	assert.Equal(t, Pt{3 * (BrickPixelSize + BrickMarginPixelSize), PlayAreaHeight - 3*BrickPixelSize - 2*BrickMarginPixelSize}, CanonicalPosToPixelPos(Pt{3, 2}))
}

func TestPixelPosToCanonicalPixelPos(t *testing.T) {
	RSeed(0)

	// Ridiculous check, but check.
	for range 10 {
		pixelPos := RPos()
		expected := CanonicalPosToPixelPos(PixelPosToCanonicalPos(pixelPos))
		assert.Equal(t, expected, PixelPosToCanonicalPixelPos(pixelPos))
	}
	assert.Equal(t, true, true)
}

func TestGetObstacles(t *testing.T) {
	RSeed(0)

	{
		var w World
		w.NextBrickId = 1
		w.Bricks = append(w.Bricks, w.NewBrick(RPos(), 13))
		buffer := make([]Rectangle, 100)
		w.GetObstacles(&w.Bricks[0], IncludingTop, &buffer)
		require.Equal(t, 4, len(buffer))
	}

	{
		var w World
		w.NextBrickId = 1
		for i := range 10 {
			w.Bricks = append(w.Bricks, w.NewBrick(RPos(), int64(i+1)))
		}

		b := w.NewBrick(RPos(), 3)

		// Check that top is included/excluded.
		buffer1 := make([]Rectangle, 100)
		w.GetObstacles(&b, IncludingTop, &buffer1)
		buffer2 := make([]Rectangle, 100)
		w.GetObstacles(&b, ExceptTop, &buffer2)
		require.Equal(t, 1, len(buffer1)-len(buffer2))

		// Check that the number of obstacles is correct.
		require.Equal(t, 10+4-1, len(buffer1))
	}

	// Check that partners for chained bricks are excluded.
	for range 10 {
		var w World
		w.NextBrickId = 1
		for i := range 10 {
			w.Bricks = append(w.Bricks, w.NewBrick(RPos(), int64(i+1)))
		}

		// Chain two bricks.
		// Choose one to be the leader.
		b1 := &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
		w.SetBrickPos(b1, RPosLeader())

		// Choose one that will be the follower.
		var b2 *Brick
		for {
			b2 = &w.Bricks[RInt(0, int64(len(w.Bricks)-1))]
			if b1 != b2 {
				break
			}
		}
		// Set the position of the follower to be suitable for chaining.
		b2Pos := b1.PixelPos
		if RInt(0, 1) == 0 {
			b2Pos.X += BrickPixelSize + BrickMarginPixelSize
		} else {
			b2Pos.Y -= BrickPixelSize + BrickMarginPixelSize
		}
		w.SetBrickPos(b2, b2Pos)

		buffer := make([]Rectangle, 100)
		w.GetObstacles(b1, IncludingTop, &buffer)
		require.True(t, slices.Contains(buffer, b2.Bounds))

		// Chain bricks.
		ChainBricks(b1, b2)

		w.GetObstacles(b1, IncludingTop, &buffer)
		require.False(t, slices.Contains(buffer, b2.Bounds))
	}
}

func TestMoveBrick(t *testing.T) {
	// Check for a simple brick.
	{
		var w World
		w.NextBrickId = 1

		var pos1 Pt
		pos1.X = RInt(0, (PlayAreaWidth-BrickPixelSize)/2)
		pos1.Y = RInt(0, PlayAreaHeight-BrickPixelSize)
		w.Bricks = append(w.Bricks, w.NewBrick(pos1, 3))

		dif := int64(137)
		pos2 := pos1.Plus(Pt{BrickPixelSize + dif, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos2, 4))

		// Move through the obstacle.
		targetPos := pos1.Plus(Pt{1000, 0})
		nMaxPixels := int64(300)
		hitObstacle := w.MoveBrick(&w.Bricks[0], targetPos, nMaxPixels,
			IgnoreObstacles)
		require.False(t, hitObstacle)

		// Stop at the obstacle.
		w.SetBrickPos(&w.Bricks[0], pos1)
		targetPos = pos1.Plus(Pt{1000, 0})
		nMaxPixels = int64(1000)
		hitObstacle = w.MoveBrick(&w.Bricks[0], targetPos, nMaxPixels,
			StopAtFirstObstacleExceptTop)
		require.True(t, hitObstacle)
		expectedPos := pos2.Minus(Pt{BrickPixelSize, 0})
		require.Equal(t, expectedPos, w.Bricks[0].PixelPos)
	}

	// Check for a chained brick.
	{
		var w World
		w.NextBrickId = 1

		var pos1 Pt
		pos1.X = RInt(0, (PlayAreaWidth-BrickPixelSize)/2)
		pos1.Y = RInt(0, PlayAreaHeight-BrickPixelSize)
		w.Bricks = append(w.Bricks, w.NewBrick(pos1, 3))

		pos2 := pos1.Plus(Pt{BrickPixelSize + BrickMarginPixelSize, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos2, 4))
		ChainBricks(&w.Bricks[0], &w.Bricks[1])

		dif := int64(137)
		pos3 := pos1.Plus(Pt{2*BrickPixelSize + BrickMarginPixelSize + dif, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos3, 4))

		// Stop at the obstacle.
		targetPos := pos1.Plus(Pt{1000, 0})
		nMaxPixels := int64(1000)
		hitObstacle := w.MoveBrick(&w.Bricks[0], targetPos, nMaxPixels,
			StopAtFirstObstacleExceptTop)
		require.True(t, hitObstacle)
		expectedPos := pos3.Minus(Pt{2*BrickPixelSize + BrickMarginPixelSize, 0})
		require.Equal(t, expectedPos, w.Bricks[0].PixelPos)
	}
}

func TestMoveBrickHelper(t *testing.T) {
	// Check for a simple brick.
	{
		var w World
		w.NextBrickId = 1

		var pos1 Pt
		pos1.X = RInt(0, (PlayAreaWidth-BrickPixelSize)/2)
		pos1.Y = RInt(0, PlayAreaHeight-BrickPixelSize)
		w.Bricks = append(w.Bricks, w.NewBrick(pos1, 3))

		dif := int64(137)
		pos2 := pos1.Plus(Pt{BrickPixelSize + dif, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos2, 4))

		targetPos := pos1.Plus(Pt{1000, 0})
		nMaxPixels := int64(1000)
		nPixelsLeft := w.MoveBrickHelper(&w.Bricks[0], targetPos, nMaxPixels,
			IncludingTop)
		require.Equal(t, dif, nMaxPixels-nPixelsLeft)
		expectedPos := pos2.Minus(Pt{BrickPixelSize, 0})
		require.Equal(t, expectedPos, w.Bricks[0].PixelPos)
	}

	// Check for a chained brick.
	{
		var w World
		w.NextBrickId = 1

		var pos1 Pt
		pos1.X = RInt(0, (PlayAreaWidth-BrickPixelSize)/2)
		pos1.Y = RInt(0, PlayAreaHeight-BrickPixelSize)
		w.Bricks = append(w.Bricks, w.NewBrick(pos1, 3))

		pos2 := pos1.Plus(Pt{BrickPixelSize + BrickMarginPixelSize, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos2, 4))
		ChainBricks(&w.Bricks[0], &w.Bricks[1])

		dif := int64(137)
		pos3 := pos1.Plus(Pt{2*BrickPixelSize + BrickMarginPixelSize + dif, 0})
		w.Bricks = append(w.Bricks, w.NewBrick(pos3, 5))

		targetPos := pos1.Plus(Pt{1000, 0})
		nMaxPixels := int64(1000)
		nPixelsLeft := w.MoveBrickHelper(&w.Bricks[0], targetPos, nMaxPixels,
			IncludingTop)
		require.Equal(t, dif, nMaxPixels-nPixelsLeft)
		expectedPos := pos3.Minus(Pt{2*BrickPixelSize + BrickMarginPixelSize, 0})
		require.Equal(t, expectedPos, w.Bricks[0].PixelPos)
	}
}
