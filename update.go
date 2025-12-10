package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gopkg.in/yaml.v3"
	"slices"
)

func (g *Gui) Update() error {
	if g.folderWatcher1.FolderContentsChanged() {
		g.LoadGuiData()
	}

	g.pointer = g.GetPointerState()
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
	if g.JustPressed(playScreenMenuButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
}

func (g *Gui) UpdatePlayScreen() {
	if g.JustPressed(homeScreenMenuButton) {
		g.state = PausedScreen
		return
	}

	// Get the player input.
	var input PlayerInput
	input.JustPressed = g.pointer.JustPressed
	input.JustReleased = g.pointer.JustReleased
	input.Pos = g.ScreenToWorld(g.pointer.Pos)
	if g.JustPressedKey(ebiten.KeyEscape) {
		g.state = PausedScreen
		return
	}
	if g.JustPressedKey(ebiten.KeyR) {
		input.ResetWorld = true
	}
	if g.JustPressedKey(ebiten.KeyC) {
		input.TriggerComingUp = true
	}

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
		g.visWorld.Step(&g.world)

		// Save best score if it got increased.
		if g.world.Score > g.BestScore {
			g.BestScore = g.world.Score
			g.uploadUserDataChannel <- g.UserData
		}

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
	if g.JustPressed(pausedScreenContinueButton1) ||
		g.JustPressed(pausedScreenContinueButton2) ||
		g.JustPressedKey(ebiten.KeyEscape) {
		g.state = PlayScreen
	}
	if g.JustPressed(pausedScreenRestartButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
	if g.JustPressed(pausedScreenHomeButton) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameOverScreen() {
	if g.JustPressed(gameOverScreenRestartButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
	if g.JustPressed(gameOverScreenHomeButton) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdateGameWonScreen() {
	if g.JustPressed(gameWonScreenRestartButton) {
		g.world = NewWorldFromPlaythrough(g.playthrough)
		g.state = PlayScreen
	}
	if g.JustPressed(gameWonScreenHomeButton) {
		g.state = HomeScreen
	}
}

func (g *Gui) UpdatePlayback() {
	nFrames := int64(len(g.playthrough.History))
	pos := g.pointer.Pos.Minus(g.horizontalDebugArea.Min)

	userRequestedPlaybackPause := g.JustPressedKey(ebiten.KeySpace) ||
		g.pointer.JustPressed && debugPlayButton.ContainsPt(pos)
	if userRequestedPlaybackPause {
		g.playbackPaused = !g.playbackPaused
	}

	// Choose target frame.
	targetFrameIdx := g.frameIdx

	// Compute the target frame index based on where on the play bar the user
	// pressed.
	if g.pointer.Pressed && debugPlayBar.ContainsPt(pos) {
		// Get the distance between the start and the cursor on the play bar.
		dx := pos.X - debugPlayBar.Min.X
		targetFrameIdx = dx * nFrames / debugPlayBar.Width()
	}

	if g.JustPressedKey(ebiten.KeyLeft) && g.IsPressed(ebiten.KeyAlt) {
		targetFrameIdx -= g.FrameSkipAltArrow
	}

	if g.JustPressedKey(ebiten.KeyRight) && g.IsPressed(ebiten.KeyAlt) {
		targetFrameIdx += g.FrameSkipAltArrow
	}

	if g.IsPressed(ebiten.KeyLeft) && g.IsPressed(ebiten.KeyShift) {
		targetFrameIdx -= g.FrameSkipShiftArrow
	}

	if g.IsPressed(ebiten.KeyRight) && g.IsPressed(ebiten.KeyShift) {
		targetFrameIdx += g.FrameSkipShiftArrow
	}

	if g.IsPressed(ebiten.KeyLeft) &&
		!g.IsPressed(ebiten.KeyShift) &&
		!g.IsPressed(ebiten.KeyAlt) {
		if g.playbackPaused {
			targetFrameIdx -= g.FrameSkipArrow
		} else {
			targetFrameIdx -= g.FrameSkipArrow * 2
		}
	}

	if g.IsPressed(ebiten.KeyRight) &&
		!g.IsPressed(ebiten.KeyShift) &&
		!g.IsPressed(ebiten.KeyAlt) {
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
	// Set virtual pointer position so that the virtual pointer can be drawn
	// in Draw().
	g.virtualPointerPos = g.WorldToScreen(input.Pos)

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
	if g.frameIdx < int64(len(g.playthrough.History)) {
		input = g.playthrough.History[g.frameIdx]
		// Set virtual pointer position so that the virtual pointer can be drawn
		// in Draw().
		g.virtualPointerPos = g.WorldToScreen(input.Pos)
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

func (g *Gui) IsPressed(k ebiten.Key) bool {
	return slices.Contains(g.pressedKeys, k)
}

func (g *Gui) JustPressedKey(k ebiten.Key) bool {
	return slices.Contains(g.justPressedKeys, k)
}

func (g *Gui) JustPressed(b Rectangle) bool {
	if !g.pointer.JustPressed {
		return false
	}

	// The b rectangle is relative to the game area.
	return b.ContainsPt(g.ScreenToGame(g.pointer.Pos))
}

func (g *Gui) GetPointerState() PointerState {
	// Check for justPressed.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return PointerState{true, true, false, Pt{int64(x), int64(y)}}
	}

	touchIDs := inpututil.AppendJustPressedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return PointerState{true, true, false, Pt{int64(x), int64(y)}}
	}

	// Check for justReleased.
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return PointerState{false, false, true, Pt{int64(x), int64(y)}}
	}

	touchIDs = inpututil.AppendJustReleasedTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return PointerState{false, false, true, Pt{int64(x), int64(y)}}
	}

	// Check for pressed.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return PointerState{true, false, false, Pt{int64(x), int64(y)}}
	}

	touchIDs = ebiten.AppendTouchIDs([]ebiten.TouchID{})
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return PointerState{true, false, false, Pt{int64(x), int64(y)}}
	}

	// Nothing is pressed, just pressed or just released.
	// Set x, y to the mouse position. This will return 0, 0 on mobile but the
	// button position should not be used by anything on the mobile if nothing
	// is pressed.
	x, y := ebiten.CursorPosition()
	return PointerState{false, false, false, Pt{int64(x), int64(y)}}
}

func LoadUserData(username string) (data UserData) {
	s := GetUserDataHttp(username)
	err := yaml.Unmarshal([]byte(s), &data)
	Check(err)
	return
}

func UploadUserData(username string, ch chan UserData) {
	for {
		// Receive a struct from the channel.
		// Blocks until a struct is received.
		data := <-ch

		// Upload the data.
		bytes, err := yaml.Marshal(data)
		Check(err)
		SetUserDataHttp(username, string(bytes))
	}
}
