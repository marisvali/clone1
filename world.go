package main

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
	Dragging        int
	DraggingOrigin  Pt
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
	w.Dragging = -1
	return w
}

func (w *World) PixelSize() (sz Pt) {
	sz.X = w.NCols*w.BrickPixelSize + (w.NCols+1)*w.MarginPixelSize
	sz.Y = w.NRows*w.BrickPixelSize + (w.NRows+1)*w.MarginPixelSize
	return
}

func (w *World) MatPosToPixelsPos(matPos Pt) (pixelPos Pt) {
	l := w.BrickPixelSize + w.MarginPixelSize
	pixelPos.X = matPos.X*l + w.MarginPixelSize
	pixelPos.Y = playHeight - (matPos.Y+1)*l
	return
}

func (w *World) Step() {

}
