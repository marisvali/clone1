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
	case HomeScreen:
		g.UpdateHomeScreen()
	case PlayScreen:
		g.UpdatePlayScreen()
	case PausedScreen:
		g.UpdatePausedScreen()
	case GameOverScreen:
		g.UpdateGameOverScreen()
	case GameWonScreen:
		g.UpdateGameWonScreen()
	case Playback:
		g.UpdatePlayback()
	case DebugCrash:
		g.UpdateDebugCrash()
	default:
		panic("unhandled default case")
	}

	return nil
}

func (g *Gui) UpdateHomeScreen() {
	if g.JustClicked(playScreenMenuButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
}

func (g *Gui) UpdatePlayScreen() {
	if g.JustClicked(homeScreenMenuButton) {
		g.state = PausedScreen
		return
	}

	// Get the player input.
	var input PlayerInput
	_, input.JustPressed, input.JustReleased, input.Pos.X, input.Pos.Y =
		g.ButtonState()
	input.Pos = g.ScreenToPlayArea(input.Pos)
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	if slices.Contains(justPressedKeys, ebiten.KeyEscape) {
		g.state = PausedScreen
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
		g.state = GameOverScreen
	}
	if g.world.State == Won {
		g.state = GameWonScreen
	}
}

func (g *Gui) UpdatePausedScreen() {
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	if g.JustClicked(pausedScreenContinueButton1) ||
		g.JustClicked(pausedScreenContinueButton2) ||
		slices.Contains(justPressedKeys, ebiten.KeyEscape) {
		g.state = PlayScreen
	}
	if g.JustClicked(pausedScreenRestartButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
	if g.JustClicked(pausedScreenHomeButton) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameOverScreen() {
	if g.JustClicked(gameOverScreenRestartButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
	if g.JustClicked(gameOverScreenHomeButton) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameWonScreen() {
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
	if g.PressedOn(g.buttonPlaybackBar) {
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

func (g *Gui) Pressed(key ebiten.Key) bool {
	return slices.Contains(g.pressedKeys, key)
}

func (g *Gui) JustPressed(key ebiten.Key) bool {
	return slices.Contains(g.justPressedKeys, key)
}

func ImageRectContainsPt(r image.Rectangle, pt image.Point) bool {
	return pt.X >= r.Min.X && pt.X <= r.Max.X && pt.Y >= r.Min.Y && pt.Y <= r.Max.Y
}

func (g *Gui) JustClicked(button Rectangle) bool {
	_, justPressed, _, x, y := g.ButtonState()
	if justPressed {
		return button.ContainsPt(Pt{x, y}.Minus(g.gameAreaOrigin))
	}
	return false
}

func (g *Gui) PressedOn(button image.Rectangle) bool {
	pressed, _, _, x, y := g.ButtonState()
	if pressed {
		return ImageRectContainsPt(button, image.Pt(int(x), int(y)))
	}
	return false
}

func (g *Gui) ButtonState() (pressed, justPressed, justReleased bool, x, y int64) {
	// Check for justPressed.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		xi, yi := ebiten.CursorPosition()
		return true, true, false, int64(xi), int64(yi)
	}

	touchIDs := inpututil.AppendJustPressedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		xi, yi := ebiten.TouchPosition(touchIDs[0])
		return true, true, false, int64(xi), int64(yi)
	}

	// Check for justReleased.
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		xi, yi := ebiten.CursorPosition()
		return false, false, true, int64(xi), int64(yi)
	}

	touchIDs = inpututil.AppendJustReleasedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		xi, yi := ebiten.TouchPosition(touchIDs[0])
		return false, false, true, int64(xi), int64(yi)
	}

	// Check for pressed.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		xi, yi := ebiten.CursorPosition()
		return true, false, false, int64(xi), int64(yi)
	}

	touchIDs = ebiten.AppendTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		xi, yi := ebiten.TouchPosition(touchIDs[0])
		return true, false, false, int64(xi), int64(yi)
	}

	// Nothing is pressed, just pressed or just released.
	// Set x, y to the mouse position. This will return 0, 0 on mobile but the
	// button position should not be used by anything on the mobile if nothing
	// is pressed.
	xi, yi := ebiten.CursorPosition()
	return false, false, false, int64(xi), int64(yi)
}
