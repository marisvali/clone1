package main

import "math"

// World rules
// - Each brick belongs to a position in the matrix.
// - The position in the matrix to which the brick belongs to is determined like
// this: the matrix position for which the center is the closest to the center
// of a brick.
// - If a brick has no brick underneath it, it falls until it reaches a position
// in the matrix where there is a brick underneath it.
// - A brick has no brick underneath it if there is no brick at the matrix
// position underneath it for X seconds.
// - While bricks move, a brick cannot overlap another brick.
// - If a brick has no brick underneath it and it falls and that would mean it
// would overlap another brick which is currently being dragged, it pushes the
// dragged brick away.

type Brick struct {
	Val       int
	PosMat    Pt
	PosPixels Pt
}

type World struct {
	NCols           int
	NRows           int
	BrickPixelSize  int
	MarginPixelSize int
	Bricks          []Brick
	Dragging        *Brick
	DraggingOrigin  Pt
}

type PlayerInput struct {
	Pos          Pt
	JustPressed  bool
	JustReleased bool
}

func NewWorld() (w World) {
	w.NCols = 6
	w.NRows = 8
	w.MarginPixelSize = 30
	w.BrickPixelSize = (playWidth - (w.MarginPixelSize * (w.NCols + 1))) / w.NCols

	w.Bricks = append(w.Bricks, Brick{
		Val:       3,
		PosMat:    Pt{3, 4},
		PosPixels: w.MatPosToPixelsPos(Pt{3, 4}),
	})
	w.Bricks = append(w.Bricks, Brick{
		Val:       2,
		PosMat:    Pt{0, 2},
		PosPixels: w.MatPosToPixelsPos(Pt{0, 2}),
	})
	w.Bricks = append(w.Bricks, Brick{
		Val:       1,
		PosMat:    Pt{5, 1},
		PosPixels: w.MatPosToPixelsPos(Pt{5, 1}),
	})
	w.Dragging = nil
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) PixelsPosToMatPos(pixelPos Pt) (matPos Pt) {
	l := float64(w.BrickPixelSize + w.MarginPixelSize)
	matPos.X = int(math.Round(float64(pixelPos.X-w.MarginPixelSize) / l))
	matPos.Y = int(math.Round(float64(playHeight-pixelPos.Y)/l - 1))
	return
}

func (w *World) MatPosToPixelsPos(matPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = matPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (matPos.Y+1)*l
	return
}

func (w *World) Step(input PlayerInput) {
	if input.JustPressed {
		// It should not be possible to be dragging anything already.
		if w.Dragging != nil {
			panic("wrong!")
		}

		// Check if there's any brick under the click.
		for i := range w.Bricks {
			p := w.Bricks[i].PosPixels
			brickSize := Pt{w.BrickPixelSize, w.BrickPixelSize}
			r := Rectangle{p, p.Plus(brickSize)}
			if r.ContainsPt(input.Pos) {
				w.Dragging = &w.Bricks[i]
				w.DraggingOrigin = input.Pos
				break
			}
		}
	}

	if input.JustReleased {
		if w.Dragging != nil {
			// Reset dragged brick's position.
			w.Dragging.PosMat = w.PixelsPosToMatPos(w.Dragging.PosPixels)
			w.Dragging.PosPixels = w.MatPosToPixelsPos(w.Dragging.PosMat)
		}
		w.Dragging = nil
	}

	if w.Dragging != nil {
		// Update dragged brick's position.
		oldPos := w.Dragging.PosPixels
		offset := input.Pos.Minus(w.DraggingOrigin)
		w.Dragging.PosPixels = w.MatPosToPixelsPos(w.Dragging.PosMat).
			Plus(offset)

		// Check if the new pos is valid.
		r := w.BrickBounds(*w.Dragging)
		newPosIsValid := true
		for i := range w.Bricks {
			if w.Dragging == &w.Bricks[i] {
				continue
			}
			r2 := w.BrickBounds(w.Bricks[i])
			if r.Intersects(r2) {
				newPosIsValid = false
				break
			}
		}

		// Revert to old pos if the new pos is invalid.
		if !newPosIsValid {
			w.Dragging.PosPixels = oldPos
		}
	}
}

func (w *World) BrickBounds(b Brick) (r Rectangle) {
	r.Corner1 = b.PosPixels
	r.Corner2 = b.PosPixels
	r.Corner2.X += w.BrickPixelSize
	r.Corner2.Y += w.BrickPixelSize
	return
}
