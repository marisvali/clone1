package main

type Pt struct {
	X int64
	Y int64
}

func (p Pt) SquaredDistTo(other Pt) int64 {
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

func (p Pt) Times(multiply int64) Pt {
	return Pt{p.X * multiply, p.Y * multiply}
}

func (p Pt) DivBy(divide int64) Pt {
	return Pt{p.X / divide, p.Y / divide}
}

func (p Pt) SquaredLen() int64 {
	return p.X*p.X + p.Y*p.Y
}

func (p Pt) To(other Pt) Pt {
	return Pt{other.X - p.X, other.Y - p.Y}
}

func (p Pt) Dot(other Pt) int64 {
	return p.X*other.X + p.Y*other.Y
}
