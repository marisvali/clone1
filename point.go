package main

type Pt struct {
	X int
	Y int
}

func (p Pt) SquaredDistTo(other Pt) int {
	return p.To(other).SquaredLen()
}

func (p *Pt) Add(other Pt) {
	p.X = p.X + other.X
	p.Y = p.Y + other.Y
}

func (p Pt) Plus(other Pt) Pt {
	return Pt{p.X + other.X, p.Y + other.Y}
}

func (p Pt) Minus(other Pt) Pt {
	return Pt{p.X - other.X, p.Y - other.Y}
}

func (p *Pt) Subtract(other Pt) {
	p.X = p.X - other.X
	p.Y = p.Y - other.Y
}

func (p Pt) Times(multiply int) Pt {
	return Pt{p.X * multiply, p.Y * multiply}
}

func (p Pt) DivBy(divide int) Pt {
	return Pt{p.X / divide, p.Y / divide}
}

func (p Pt) SquaredLen() int {
	return p.X*p.X + p.Y*p.Y
}

func (p Pt) To(other Pt) Pt {
	return Pt{other.X - p.X, other.Y - p.Y}
}

func (p Pt) Dot(other Pt) int {
	return p.X*other.X + p.Y*other.Y
}
