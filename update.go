package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Gui) Update() error {
	var input PlayerInput
	input.Pos.X, input.Pos.Y = ebiten.CursorPosition()
	input.Pos = g.screenToPlayCoord(input.Pos)
	input.JustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	input.JustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	g.world.Step(input)
	return nil
}
