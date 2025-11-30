package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"image"
	"slices"
)

func (g *Gui) Update() error {
	g.pressedKeys = g.pressedKeys[:0]
	g.pressedKeys = inpututil.AppendPressedKeys(g.pressedKeys)
	g.justPressedKeys = g.justPressedKeys[:0]
	g.justPressedKeys = inpututil.AppendJustPressedKeys(g.justPressedKeys)

	switch g.state {
	case GameOngoing:
		g.UpdateGameOngoing()
	case Playback:
		g.UpdatePlayback()
	case DebugCrash:
		g.UpdateDebugCrash()
	case HomeScreen:
		g.UpdateHomeScreen()
	case GamePaused:
		g.UpdateGamePaused()
	case GameOver:
		g.UpdateGameOver()
	case GameWon:
		g.UpdateGameWon()
	default:
		panic("unhandled default case")
	}

	return nil
}

func (g *Gui) UpdateGameOngoing() {
	// Get the player input.
	var input PlayerInput

	input.JustPressed = false
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		input.JustPressed = true
	}
	touchIDs := inpututil.AppendJustPressedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		input.JustPressed = true
	}

	input.JustReleased = false
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		input.JustReleased = true

	}
	touchIDs = inpututil.AppendJustReleasedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		input.JustReleased = true
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		input.Pos = Pt{int64(x), int64(y)}
	}
	touchIDs = ebiten.AppendTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		input.Pos = Pt{int64(x), int64(y)}
	}

	input.Pos = g.screenToPlayArea(input.Pos)
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	if slices.Contains(justPressedKeys, ebiten.KeyEscape) {
		g.state = GamePaused
		return
	}
	if slices.Contains(justPressedKeys, ebiten.KeyR) {
		input.ResetWorld = true
	}
	if slices.Contains(justPressedKeys, ebiten.KeyC) {
		input.TriggerComingUp = true
	}

	// Remember cursor position in order to draw the virtual cursor during
	// Draw().
	g.mousePt = input.Pos

	// We want to slow down the game sometimes by only updating the World once
	// every n frames. This is very useful when it's necessary to do some tricky
	// moves in order to trigger an edge case (e.g. drag brick A on top of brick
	// B while brick C is falling on B). It's hard to do at regular speed and if
	// we modify the speeds and accelerations within the World, the test isn't
	// really performed under production conditions.
	//
	// If the game is slowed down, remember clicks and key presses that happen
	// during frames where we don't update the World, so that they can be sent
	// to the World in the next frame where the World is updated.
	if input.EventOccurred() {
		// When the player clicks something, we remember the click and the
		// position.
		g.accumulatedInput = input
	} else {
		// We remember the last mouse position, but only if there isn't a click
		// recorded already in g.accumulatedInput. If a click was recorded in
		// g.accumulatedInput, we want to remember the Pos at which the click
		// occurred.
		if !g.accumulatedInput.EventOccurred() {
			g.accumulatedInput.Pos = input.Pos
		}
	}
	if g.frameIdx%g.slowdownFactor == 0 {
		// Save the input in the playthrough.
		g.playthrough.History = append(g.playthrough.History, g.accumulatedInput)
		if g.recordingFile != "" {
			// IMPORTANT: save the playthrough before stepping the World. If
			// a bug in the World causes it to crash, we want to save the input
			// that caused the bug before the program crashes.
			// WriteFile(g.recordingFile, g.playthrough.Serialize())
		}

		// Step the world.
		g.world.Step(g.accumulatedInput)
		g.accumulatedInput = PlayerInput{}
	}

	// Finally increase the frame.
	g.frameIdx++

	if g.world.State == Lost {
		g.state = GameOver
	}
	if g.world.State == Won {
		g.state = GameWon
	}
}

func (g *Gui) Pressed(key ebiten.Key) bool {
	return slices.Contains(g.pressedKeys, key)
}

func (g *Gui) JustPressed(key ebiten.Key) bool {
	return slices.Contains(g.justPressedKeys, key)
}

func ImageRectContainsPt(r image.Rectangle, pt image.Point) bool {
	return pt.X >= r.Min.X && pt.X <= r.Max.X && pt.Y >= r.Min.Y && pt.Y <= r.Max.Y
}

func (g *Gui) JustClicked(button image.Rectangle) bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		x, y := ebiten.CursorPosition()
		return ImageRectContainsPt(button, image.Pt(x, y))
	}
	touchIDs := inpututil.AppendJustPressedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) != 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return ImageRectContainsPt(button, image.Pt(x, y))
	}
	return false
}

func (g *Gui) LeftClickPressedOn(button image.Rectangle) bool {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
		x, y := ebiten.CursorPosition()
		return ImageRectContainsPt(button, image.Pt(x, y))
	}
	touchIDs := ebiten.AppendTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) != 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return ImageRectContainsPt(button, image.Pt(x, y))
	}
	return false
}

func (g *Gui) UpdatePlayback() {
	nFrames := int64(len(g.playthrough.History))

	userRequestedPlaybackPause := g.JustPressed(ebiten.KeySpace) || g.JustClicked(g.buttonPlaybackPlay)
	if userRequestedPlaybackPause {
		g.playbackPaused = !g.playbackPaused
	}

	// Choose target frame.
	targetFrameIdx := g.frameIdx

	// Compute the target frame index based on where on the play bar the user
	// clicked.
	if g.LeftClickPressedOn(g.buttonPlaybackBar) {
		// Get the distance between the start and the cursor on the play bar.
		x, _ := ebiten.CursorPosition()
		dx := int64(x - g.buttonPlaybackBar.Min.X)
		targetFrameIdx = dx * nFrames / int64(g.buttonPlaybackBar.Size().X)
	}

	if g.JustPressed(ebiten.KeyLeft) && g.Pressed(ebiten.KeyAlt) {
		targetFrameIdx -= g.FrameSkipAltArrow
	}

	if g.JustPressed(ebiten.KeyRight) && g.Pressed(ebiten.KeyAlt) {
		targetFrameIdx += g.FrameSkipAltArrow
	}

	if g.Pressed(ebiten.KeyLeft) && g.Pressed(ebiten.KeyShift) {
		targetFrameIdx -= g.FrameSkipShiftArrow
	}

	if g.Pressed(ebiten.KeyRight) && g.Pressed(ebiten.KeyShift) {
		targetFrameIdx += g.FrameSkipShiftArrow
	}

	if g.Pressed(ebiten.KeyLeft) && !g.Pressed(ebiten.KeyShift) && !g.Pressed(ebiten.KeyAlt) {
		if g.playbackPaused {
			targetFrameIdx -= g.FrameSkipArrow
		} else {
			targetFrameIdx -= g.FrameSkipArrow * 2
		}
	}

	if g.Pressed(ebiten.KeyRight) && !g.Pressed(ebiten.KeyShift) && !g.Pressed(ebiten.KeyAlt) {
		targetFrameIdx += g.FrameSkipArrow
	}

	if targetFrameIdx < 0 {
		targetFrameIdx = 0
	}

	if targetFrameIdx >= nFrames {
		targetFrameIdx = nFrames - 1
	}

	if targetFrameIdx != g.frameIdx {
		// Rewind.
		g.world = NewWorldFromPlaythrough(g.playthrough)

		// Replay the world.
		for i := int64(0); i < targetFrameIdx; i++ {
			g.world.Step(g.playthrough.History[i])
		}

		// Set the current frame idx.
		g.frameIdx = targetFrameIdx
	}

	// Get input from recording.
	input := g.playthrough.History[g.frameIdx]
	// Remember cursor position in order to draw the virtual cursor during
	// Draw().
	g.mousePt = input.Pos

	// input = g.ai.Step(&g.world)
	if !g.playbackPaused {
		g.world.Step(input)

		if g.frameIdx < nFrames-1 {
			g.frameIdx++
		}
	}

	if g.world.AssertionFailed {
		g.playbackPaused = true
	}
}

func (g *Gui) UpdateDebugCrash() {
	var input PlayerInput
	// Remember cursor position in order to draw the virtual cursor during
	// Draw().
	if g.frameIdx < int64(len(g.playthrough.History)) {
		input = g.playthrough.History[g.frameIdx]
		g.mousePt = input.Pos
	}

	// Don't do anything, wait for the player to press a key.
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)

	// Go to the next frame.
	goToNextFrame := slices.Contains(justPressedKeys, ebiten.KeyD) ||
		slices.Contains(justPressedKeys, ebiten.KeyRight)
	if goToNextFrame && g.frameIdx < int64(len(g.playthrough.History)) {
		g.world.Step(input)
		g.frameIdx++
	}

	// Go to the previous frame.
	goToPreviousFrame := slices.Contains(justPressedKeys, ebiten.KeyA) ||
		slices.Contains(justPressedKeys, ebiten.KeyLeft)
	if goToPreviousFrame && g.frameIdx > 0 {
		g.frameIdx--

		// I have no better way to go to the previous frame than redoing all the
		// frames from the beginning.
		g.world = NewWorldFromPlaythrough(g.playthrough)
		for i := range g.frameIdx {
			g.world.Step(g.playthrough.History[i])
		}
	}
}

func (g *Gui) UpdateHomeScreen() {
	if g.JustClicked(g.buttonPlay) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = GameOngoing
	}
}

func (g *Gui) UpdateGamePaused() {
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	if g.JustClicked(g.buttonContinue) ||
		slices.Contains(justPressedKeys, ebiten.KeyEscape) {
		g.state = GameOngoing
	}
	if g.JustClicked(g.buttonRestart) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = GameOngoing
	}
	if g.JustClicked(g.buttonHome) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameOver() {
	if g.JustClicked(g.buttonRestart) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = GameOngoing
	}
	if g.JustClicked(g.buttonHome) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameWon() {
}
