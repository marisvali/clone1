package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"slices"
)

func (g *Gui) Update() error {
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

func (g *Gui) UpdatePlayback() {
	// Don't do anything if we reached the end, don't allow anymore updates on
	// the world.
	if g.frameIdx < int64(len(g.playthrough.History)) {
		// Get the input from the playthrough.
		input := g.playthrough.History[g.frameIdx]

		// Remember cursor position in order to draw the virtual cursor during
		// Draw().
		g.mousePt = input.Pos
		// Step the world.
		g.world.Step(input)
		// Finally increase the frame.
		g.frameIdx++
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
