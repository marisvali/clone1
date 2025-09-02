package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image"
	"image/color"
)

func (g *Gui) Draw(screen *ebiten.Image) {
	// The screen bitmap has the aspect ratio of the game window. We fill it
	// with some background. Then, we decide on a play region inside of screen,
	// on which we draw all the actually interesting elements of our game.
	screen.Fill(color.NRGBA{
		R: 180,
		G: 180,
		B: 180,
		A: 255,
	})

	marginX := (screen.Bounds().Size().X - playWidth) / 2
	marginY := (screen.Bounds().Size().Y - playHeight) / 2

	play := SubImage(screen, image.Rect(
		marginX,
		marginY,
		marginX+playWidth,
		marginY+playHeight))

	// Highlight the play region for dev purposes.
	play.Fill(color.NRGBA{
		R: 230,
		G: 237,
		B: 240,
		A: 255,
	})

	// Draw empty spaces.
	for y := 0; y < g.world.NRows; y++ {
		for x := 0; x < g.world.NCols; x++ {
			pos := g.world.MatPosToPixelsPos(Pt{x, y})
			DrawSprite(play, g.imgBlank, float64(pos.X), float64(pos.Y),
				float64(g.world.BrickPixelSize),
				float64(g.world.BrickPixelSize))

			brickRegion := SubImage(play, image.Rect(pos.X, pos.Y,
				pos.X+g.world.BrickPixelSize,
				pos.Y+g.world.BrickPixelSize))
			g.drawText(brickRegion, fmt.Sprintf("%dx%d", x, y), true,
				true,
				color.NRGBA{
					R: 0,
					G: 100,
					B: 0,
					A: 255,
				})
		}
	}

	// Draw actual bricks.
	for _, b := range g.world.Bricks {
		pos := b.PosPixels
		var img *ebiten.Image
		switch b.Val {
		case 1:
			img = g.img1
		case 2:
			img = g.img2
		case 3:
			img = g.img3
		}
		DrawSprite(play, img, float64(pos.X), float64(pos.Y),
			float64(g.world.BrickPixelSize),
			float64(g.world.BrickPixelSize))

		// brickRegion := SubImage(play, image.Rect(pos.X, pos.Y,
		// 	pos.X+g.world.BrickPixelSize,
		// 	pos.Y+g.world.BrickPixelSize))
		// brickRegion.Fill(color.NRGBA{
		// 	R: 0,
		// 	G: 0,
		// 	B: 0,
		// 	A: 255,
		// })
		// g.drawText(brickRegion, fmt.Sprintf("%d", b.Val), true,
		// 	true,
		// 	color.NRGBA{
		// 		R: 0,
		// 		G: 100,
		// 		B: 0,
		// 		A: 255,
		// 	})
	}

	g.drawText(screen, fmt.Sprintf("ActualTPS: %f", ebiten.ActualTPS()), false,
		false,
		color.NRGBA{
			R: 0,
			G: 100,
			B: 0,
			A: 255,
		})
	// dx, dy := ebiten.Wheel()
	// ebitenutil.DebugPrint(screen, fmt.Sprintf("dx: %f dy: %f", dx, dy))
}

func (g *Gui) drawText(screen *ebiten.Image, message string, centerX bool, centerY bool, color color.Color) {
	// Remember that text there is an origin point for the text.
	// That origin point is kind of the lower-left corner of the bounds of the
	// text. Kind of. Read the BoundString docs to understand, particularly this
	// image:
	// https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyphterms_2x.png
	// This means that if you do text.Draw at (x, y), most of the text will
	// appear above y, and a little bit under y. If you want all the pixels in
	// your text to be above y, you should do text.Draw at
	// (x, y - text.BoundString().Max.Y).
	textSize := text.BoundString(g.defaultFont, message)
	var offsetX int
	if centerX {
		offsetX = (screen.Bounds().Dx() - textSize.Dx()) / 2
	} else {
		offsetX = 0
	}

	var offsetY int
	if centerY {
		offsetY = (screen.Bounds().Dy() - textSize.Dy()) / 2
	} else {
		offsetY = 0
	}

	textX := screen.Bounds().Min.X + offsetX
	textY := screen.Bounds().Max.Y - offsetY - textSize.Max.Y
	text.Draw(screen, message, g.defaultFont, textX, textY, color)
}

func (g *Gui) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// I receive the game window's actual width and height, via
	// outsideWidth, outsideHeight. I have to return the size I want, in pixels,
	// for the bitmap that will be drawn in the window.
	//
	// The way it works:
	// - I can return any size I want.
	// - In the Draw method, I will receive the screen bitmap, which will have
	// the size in pixels that I return here.
	// - The screen bitmap from Draw method will be scaled automatically by
	// ebitengine to fit inside the window.
	// - The scaling will preserve the aspect ratio of the screen bitmap. Which
	// means that black bars will appear left and right or top and bottom if
	// the aspect ratio of the screen bitmap is different from the aspect ratio
	// of the window.
	//
	// What I want (for this game):
	// - Cover the entire window with some background, even if the
	// interesting parts are only in some area in the center.
	// - Have a "play region" that I can reason about easily, no matter the
	// aspect ratio or the resolution of the user's screen.
	// - Have a "play region" that makes sense for a vertical smartphone. So,
	// taller than wider. But don't worry too much about matching the exact
	// aspect ratio of a particular smartphone model.
	// - Have a "play region" with enough pixels that most of the time
	// ebitengine will have to shrink the image, not enlarge it.
	//
	// Solution:
	// - Have a fixed "play region", for example 1200 x 2000 pixels.
	// - Compute screenWidth and screenHeight so that the aspect ratio of the
	// screen bitmap is the same as the aspect ratio of the game window. This
	// means screenWidth / screenHeight = outsideWidth / outsideHeight.
	// - Compute screenWidth and screenHeight such that the play region is as
	// large as it can be, but still fit inside the screen. This means either
	// screenWidth = 1200 or screenHeight = 2000.
	//
	// Find out if we need to match the screen width to the play width or the
	// screen height to the play height.
	// The aspect ratio of a rectangle is width / height. As an aspect ratio
	// goes down to 0, the rectangle gets progressively thinner/taller.
	// aspectRatio(rectangleA) < aspectRatio(rectangleB) means that rectangleA
	// is thinner/taller than rectangleB.
	// So if I scale rectangleB to the maximum size that fits inside rectangleA
	// then rectangleB will fill the width of rectangleA, and there will be
	// space left at the top and the bottom.
	// So, if aspectRatio(rectangleA) < aspectRatio(rectangleB), I will have
	// rectangleA.width == rectangleB.width.
	// I want play to fit inside screen, so screen is A and play is B.
	outsideAspectRatio := float64(outsideWidth) / float64(outsideHeight)
	screenAspectRatio := outsideAspectRatio
	playAspectRatio := float64(playWidth) / float64(playHeight)
	if screenAspectRatio < playAspectRatio {
		screenWidth = playWidth
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenHeight = int(float64(screenWidth) / screenAspectRatio)
	} else {
		screenHeight = playHeight
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenWidth = int(float64(screenHeight) * screenAspectRatio)
	}

	// Store these values in Gui so that Update() can use them as well,
	// otherwise only Draw() will have access to them via the size of the
	// screen parameter it receives.
	g.screenWidth = screenWidth
	g.screenHeight = screenHeight
	return
}

func (g *Gui) getWindowSize() Pt {
	playSize := Pt{g.world.NRows, g.world.NCols}.Times(g.world.BrickPixelSize + g.world.MarginPixelSize)
	windowSize := playSize
	windowSize.X += 20
	windowSize.Y += 20

	return windowSize
}

func (g *Gui) loadGuiData() {
	// Read from the disk over and over until a full read is possible.
	// This repetition is meant to avoid crashes due to reading files
	// while they are still being written.
	// It's a hack but possibly a quick and very useful one.
	// This repeated reading is only useful when we're not reading from the
	// embedded filesystem. When we're reading from the embedded filesystem we
	// want to crash as soon as possible. We might be in the browser, in which
	// case we want to see an error in the developer console instead of a page
	// that keeps trying to load and reports nothing.
	if g.FSys == nil {
		CheckCrashes = false
	}
	for {
		CheckFailed = nil
		g.imgBlank = LoadImage(g.FSys, "data/gui/blank.png")
		g.img1 = LoadImage(g.FSys, "data/gui/1.png")
		g.img2 = LoadImage(g.FSys, "data/gui/2.png")
		g.img3 = LoadImage(g.FSys, "data/gui/3.png")
		if CheckFailed == nil {
			break
		}
	}
	CheckCrashes = true

	g.updateWindowSize()
}

func (g *Gui) updateWindowSize() {
	// windowSize := g.getWindowSize()
	// ebiten.SetWindowSize(windowSize.X.ToInt(), windowSize.Y.ToInt())
	width, height := ebiten.ScreenSizeInFullscreen()
	size := min(width, height) * 8 / 10
	ebiten.SetWindowSize(size, size)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Clone1")
}

func (g *Gui) screenToPlayCoord(pt Pt) Pt {
	marginX := (g.screenWidth - playWidth) / 2
	marginY := (g.screenHeight - playHeight) / 2
	pt.X -= marginX
	pt.Y -= marginY
	return pt
}

func (g *Gui) playToScreenCoord(pt Pt) Pt {
	marginX := (g.screenWidth - playWidth) / 2
	marginY := (g.screenHeight - playHeight) / 2
	pt.X += marginX
	pt.Y += marginY
	return pt
}
