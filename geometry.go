package main

type Line struct {
	Start Pt
	End   Pt
}

type Circle struct {
	Center   Pt
	Diameter int
}

type Square struct {
	Center Pt
	Size   int
}

type Rectangle struct {
	Corner1 Pt
	Corner2 Pt
}

func Abs(x int) int {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func Min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func Max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func MinMax(x int, y int) (int, int) {
	if x < y {
		return x, y
	} else {
		return y, x
	}
}

func (r *Rectangle) Width() int {
	return Abs(r.Corner1.X - r.Corner2.X)
}

func (r *Rectangle) Height() int {
	return Abs(r.Corner1.Y - r.Corner2.Y)
}

func (r *Rectangle) Min() Pt {
	return Pt{Min(r.Corner1.X, r.Corner2.X), Min(r.Corner1.Y, r.Corner2.Y)}
}

func (r *Rectangle) Max() Pt {
	return Pt{Max(r.Corner1.X, r.Corner2.X), Max(r.Corner1.Y, r.Corner2.Y)}
}

func (r *Rectangle) ContainsPt(pt Pt) bool {
	minX, maxX := MinMax(r.Corner1.X, r.Corner2.X)
	minY, maxY := MinMax(r.Corner1.Y, r.Corner2.Y)
	return pt.X >= minX && pt.X <= maxX && pt.Y >= minY && pt.Y <= maxY
}

func (r *Rectangle) Intersects(other Rectangle) bool {
	minX1, maxX1 := MinMax(r.Corner1.X, r.Corner2.X)
	minY1, maxY1 := MinMax(r.Corner1.Y, r.Corner2.Y)
	minX2, maxX2 := MinMax(other.Corner1.X, other.Corner2.X)
	minY2, maxY2 := MinMax(other.Corner1.Y, other.Corner2.Y)
	return minX1 < maxX2 && maxX1 > minX2 && minY1 < maxY2 && maxY1 > minY2
}
