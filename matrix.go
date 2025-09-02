package main

type Mat struct {
	cells []int
	size  Pt
}

func NewMat(size Pt) Mat {
	m := Mat{}
	m.size = size
	m.cells = make([]int, size.X*size.Y)
	return m
}

func (m *Mat) Set(pos Pt, val int) {
	m.cells[pos.Y*m.size.X+pos.X] = val
}

func (m *Mat) Get(pos Pt) int {
	return m.cells[pos.Y*m.size.X+pos.X]
}

func (m *Mat) InBounds(pt Pt) bool {
	return pt.X >= 0 &&
		pt.Y >= 0 &&
		pt.Y < m.size.Y &&
		pt.X < m.size.X
}

func (m *Mat) Submat(pos, size Pt) Mat {
	sm := NewMat(size)
	i := Pt{}
	for i.Y = 0; i.Y < size.Y; i.Y++ {
		for i.X = 0; i.X < size.X; i.X++ {
			sm.Set(i, m.Get(pos.Plus(i)))
		}
	}
	return sm
}
