package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Gui) Update() error {
	// Get the player input.
	var input PlayerInput
	if g.state == Playback {
		// If we're playing back a playthrough, get the input from the
		// playthrough.
		if g.frameIdx < int64(len(g.playthrough.History)) {
			input = g.playthrough.History[g.frameIdx]
		}
	} else {
		// If we're playing back a playthrough, get the current input from the
		// player.
		x, y := ebiten.CursorPosition()
		input.Pos = Pt{int64(x), int64(y)}
		input.Pos = g.screenToPlayCoord(input.Pos)
		input.JustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		input.JustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

		// Save the input in the playthrough only if we're not playing back.
		g.playthrough.History = append(g.playthrough.History, input)
		if g.recordingFile != "" {
			// IMPORTANT: save the playthrough before stepping the World. If
			// a bug in the World causes it to crash, we want to save the input
			// that caused the bug before the program crashes.
			WriteFile(g.recordingFile, g.playthrough.Serialize())
		}
	}

	// Remember cursor position in order to draw the virtual cursor during
	// Draw().
	g.mousePt = input.Pos
	// Step the world.
	g.world.Step(input)
	// Finally increase the frame.
	g.frameIdx++
	return nil
}
