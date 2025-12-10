package main

import (
	_ "image/png"
)

// TemporaryAnimation represents an animation that appears in one place, runs
// for a while and then it goes away. It doesn't represent an ongoing entity in
// the World, it is a standalone effect, like a splash.
type TemporaryAnimation struct {
	Pos         Pt
	Animation   Animation
	NFramesLeft int64
}

// VisWorld is a world parallel to World that holds "visual logic". Its role is
// to store data and execute logic for ongoing visual effects like animations.
// Draw() relies the information in VisWorld to draw things, just like it relies
// on World.
//
// VisWorld runs parallel to World and is meant to be updated alongside World,
// in the Update() function.
type VisWorld struct {
	Animations Animations
	Temporary  []*TemporaryAnimation
}

func NewVisWorld(anims Animations) (v VisWorld) {
	v.Animations = anims
	return v
}

func (v *VisWorld) Step(w *World) {
	// Step existing animations.
	for _, a := range v.Temporary {
		a.NFramesLeft--
		a.Animation.Step()
	}

	// Filter out obsolete animations.
	n := 0
	for i := range v.Temporary {
		if v.Temporary[i].NFramesLeft > 0 {
			v.Temporary[n] = v.Temporary[i]
			n++
		}
	}
	v.Temporary = v.Temporary[:n]

	// Create new animations if necessary.
	for _, b := range w.JustMergedBricks {
		// The radial splash has its center match the brick's center.
		splashRadial := TemporaryAnimation{}
		splashRadial.Animation = v.Animations.animSplashRadial
		// One-shot animation, go through all the images once then end.
		splashRadial.NFramesLeft = splashRadial.Animation.TotalNFrames()
		splashRadial.Pos = b.Bounds.Center()
		v.Temporary = append(v.Temporary, &splashRadial)

		// The radial splash has its top-center match the brick's center.
		splashDown := TemporaryAnimation{}
		splashDown.Animation = v.Animations.animSplashDown
		// One-shot animation, go through all the images once then end.
		splashDown.NFramesLeft = splashDown.Animation.TotalNFrames()
		splashDown.Pos = b.Bounds.Center()
		splashDown.Pos.Y += b.Bounds.Height() / 2
		v.Temporary = append(v.Temporary, &splashDown)
	}
}
