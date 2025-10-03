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
	default:
		panic("unhandled default case")
	}

	return nil
}

func (g *Gui) UpdateGameOngoing() {
	// Get the player input.
	var input PlayerInput
	x, y := ebiten.CursorPosition()
	input.Pos = Pt{int64(x), int64(y)}
	input.Pos = g.screenToPlayCoord(input.Pos)
	input.JustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	input.JustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	if slices.Contains(justPressedKeys, ebiten.KeyR) {
		input.ResetWorld = true
	}
	if slices.Contains(justPressedKeys, ebiten.KeyC) {
		input.TriggerComingUp = true
	}

	// Save the input in the playthrough.
	g.playthrough.History = append(g.playthrough.History, input)
	if g.recordingFile != "" {
		// IMPORTANT: save the playthrough before stepping the World. If
		// a bug in the World causes it to crash, we want to save the input
		// that caused the bug before the program crashes.
		WriteFile(g.recordingFile, g.playthrough.Serialize())
	}

	// Remember cursor position in order to draw the virtual cursor during
	// Draw().
	g.mousePt = input.Pos
	// Step the world.
	g.world.Step(input)
	// Finally increase the frame.
	g.frameIdx++
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
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		return false
	}
	x, y := ebiten.CursorPosition()
	return ImageRectContainsPt(button, image.Pt(x, y))
}

func (g *Gui) LeftClickPressedOn(button image.Rectangle) bool {
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
		return false
	}
	x, y := ebiten.CursorPosition()
	return ImageRectContainsPt(button, image.Pt(x, y))
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

	if g.Pressed(ebiten.KeyX) {
		w1 := NewWorld()
		for i := int64(0); i < nFrames-1; i++ {
			w1.Step(g.playthrough.History[i])
		}

		w2 := NewWorld()
		for i := int64(0); i < nFrames-1; i++ {
			w2.Step(g.playthrough.History[i])
		}

		println("got here")
	}

	if targetFrameIdx < 0 {
		targetFrameIdx = 0
	}

	if targetFrameIdx >= nFrames {
		targetFrameIdx = nFrames - 1
	}

	if targetFrameIdx != g.frameIdx {
		// Rewind.
		g.world = NewWorld()

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
		g.world = NewWorld()
		for i := range g.frameIdx {
			g.world.Step(g.playthrough.History[i])
		}
	}
}
