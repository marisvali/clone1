package main

type Mat struct {
	cells []*Brick
	size  Pt
}

func NewMat(size Pt) Mat {
	m := Mat{}
	m.size = size
	m.cells = make([]*Brick, size.X*size.Y)
	return m
}

func (m *Mat) Set(pos Pt, b *Brick) {
	m.cells[pos.Y*m.size.X+pos.X] = b
}

func (m *Mat) Get(pos Pt) *Brick {
	return m.cells[pos.Y*m.size.X+pos.X]
}

func (m *Mat) Occupied(pos Pt) bool {
	return m.cells[pos.Y*m.size.X+pos.X] != nil
}

func (m *Mat) InBounds(pt Pt) bool {
	return pt.X >= 0 &&
		pt.Y >= 0 &&
		pt.Y < m.size.Y &&
		pt.X < m.size.X
}

func (m *Mat) Reset() {
	for i := range m.cells {
		m.cells[i] = nil
	}
}
