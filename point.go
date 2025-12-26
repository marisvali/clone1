package main

import "fmt"

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

// MarshalYAML turns Pt into a string.
// Useful because if I just let the YAML library do the default marshalling, it
// will turn the X and Y fields into X and "Y" because Y is shorthand for
// "yes/true" in YAML. Plus, it's shorter and easier to read two numbers in
// a single line.
// With this custom marshalling:
// Pos: [5, 19]
// Without this custom marshalling:
// Pos:
// - X: 5
// - "Y": 19
func (p Pt) MarshalYAML() ([]byte, error) {
	s := fmt.Sprintf("[ %d, %d ]", p.X, p.Y)
	return []byte(s), nil
}

func (p *Pt) UnmarshalYAML(b []byte) error {
	s := string(b)
	n, err := fmt.Sscanf(s, "[ %d, %d ]", &p.X, &p.Y)
	if n != 2 {
		Check(fmt.Errorf("failed to get exactly 2 int64 from string %s", s))
	}
	return err
}
