package main

type Tiger struct {
	Pos       Pt
	Bounds    Pt
	goingLeft bool
}

func (t *Tiger) Step(w *World) {
	if t.Pos.X <= 400 {
		t.goingLeft = false
	}
	if t.Pos.X >= 600 {
		t.goingLeft = true
	}

	if t.goingLeft {
		t.Pos.X--
	} else {
		t.Pos.X++
	}
}
