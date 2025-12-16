package main

import "fmt"

type Test struct {
	Bricks []TestBrick `yaml:"Bricks"`
}

type TestBrick struct {
	Value       int64  `yaml:"Value"`
	Pos         Pt     `yaml:"Pos"`
	Offset      Pt     `yaml:"Offset"`
	ChainedType string `yaml:"ChainedType"`
	ChainedVal  int64  `yaml:"ChainedVal"`
}

func (t *Test) GetLevel() (l Level) {
	l.TimerDisabled = true
	for _, b := range t.Bricks {
		var bp BrickParams
		bp.Val = b.Value
		bp.Pos = CanonicalPosToPixelPos(b.Pos)
		bp.Pos.Add(b.Offset)
		l.BricksParams = append(l.BricksParams, bp)

		if b.ChainedType != "" {
			bp.Val = b.ChainedVal
			if b.ChainedType == "right" {
				bp.Pos.Add(Pt{BrickPixelSize + BrickMarginPixelSize, 0})
			} else if b.ChainedType == "top" {
				bp.Pos.Add(Pt{0, -(BrickPixelSize + BrickMarginPixelSize)})
			} else {
				panic(fmt.Errorf("invalid chained type: %s", b.ChainedType))
			}

			l.BricksParams = append(l.BricksParams, bp)
			chain := ChainParams{
				Brick1: int64(len(l.BricksParams)) - 2,
				Brick2: int64(len(l.BricksParams)) - 1,
			}
			l.ChainsParams = append(l.ChainsParams, chain)
		}
	}
	return
}
