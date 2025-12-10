package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	_ "image/png"
)

// AnimationFps is the global number that says how fast animations run.
// The Update() method runs at 60 FPS (ebitengine's default). But often times
// animations do not need to be as detailed.
// This is a global setting for now, until we need animations that run at
// different rates. However, it would be inconsistent and strange. It's probably
// best to choose an FPS for all animations in a game and stick with it.
const AnimationFps = 30

// AnimationFramesPerImage is the number we actually use in many computations so
// just set it here.
const AnimationFramesPerImage = 60 / AnimationFps

// Animation represents an instance of a running animation.
// It is cheap to copy this struct. You should make copies for every
// instance of an animation that you need.
// The idea is that once the images are loaded, there's no need to change
// this data. So you can just copy around the references to the images.
type Animation struct {
	Imgs     []*ebiten.Image
	ImgIndex int64
	FrameIdx int64
}

func NewAnimation(fsys FS, name string) (a Animation) {
	count := 1
	for {
		fullName := name + "-" + fmt.Sprintf("%02d", count) + ".png"
		if !FileExists(fsys, fullName) {
			break
		}

		img := LoadImage(fsys, fullName)
		a.Imgs = append(a.Imgs, img)
		count++
	}

	// If no files exist following the format "player1.png", "player2.png" ..
	// try just loading "player.png".
	if count == 1 {
		fullName := name + ".png"
		img := LoadImage(fsys, fullName)
		a.Imgs = append(a.Imgs, img)
	}
	a.ImgIndex = 0
	return
}

func (a *Animation) Step() {
	a.FrameIdx++
	if a.FrameIdx == AnimationFramesPerImage {
		a.FrameIdx = 0
		a.ImgIndex++
	}
}

func (a *Animation) CurrentImg() *ebiten.Image {
	return a.Imgs[a.ImgIndex]
}

func (a *Animation) TotalNFrames() int64 {
	return AnimationFramesPerImage * int64(len(a.Imgs))
}
