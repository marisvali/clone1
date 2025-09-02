package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
)

// DrawSprite draws img on screen.
// x and y are in the following coordinate system:
// - The top-left pixel of screen has coordinates (0, 0).
// - The bottom-right pixel of screen has coordinates
// (screenWidth - 1, screenHeight - 1).
func DrawSprite(screen *ebiten.Image, img *ebiten.Image,
	x float64, y float64, targetWidth float64, targetHeight float64) {
	op := &ebiten.DrawImageOptions{}

	// Resize image to fit the target size we want to draw.
	// This kind of scaling is very useful during development when the final
	// sizes are not decided, and thus it's impossible to have final sprites.
	// For an actual release, scaling should be avoided.
	imgSize := img.Bounds().Size()
	newDx := targetWidth / float64(imgSize.X)
	newDy := targetHeight / float64(imgSize.Y)
	op.GeoM.Scale(newDx, newDy)
	op.GeoM.Translate(float64(screen.Bounds().Min.X)+x, float64(screen.Bounds().Min.Y)+y)
	screen.DrawImage(img, op)
}

func DrawSpriteXY(screen *ebiten.Image, img *ebiten.Image,
	x float64, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screen.Bounds().Min.X)+x, float64(screen.Bounds().Min.Y)+y)
	screen.DrawImage(img, op)
}

// SubImage returns a sub-region of screen.
// r indicates a rectangle inside of screen, in the following coordinate system:
// - The top-left pixel of screen has coordinates (0, 0).
// - The bottom-right pixel of screen has coordinates
// (screenWidth - 1, screenHeight - 1).
func SubImage(screen *ebiten.Image, r image.Rectangle) *ebiten.Image {
	// Do this because when dealing with sub-images in general I think in
	// relative coordinates. So for img2 = img1.SubImage(pt1, pt2) I now expect
	// that img2.At(0, 0) indicates the same pixel as img1.At(pt1). Ebitengine
	// doesn't do it like that. I still need to use img2.At(pt1) to indicate
	// pixel img1.At(pt1). I don't know why Ebitengine does it like that.
	// Personally, I'm used to a different style, one of the main reasons for
	// working with subimages, for me, is to be able to think in local
	// coordinates instead of global ones.
	minPt := screen.Bounds().Min
	r.Min = r.Min.Add(minPt)
	r.Max = r.Max.Add(minPt)
	return screen.SubImage(r).(*ebiten.Image)
}
