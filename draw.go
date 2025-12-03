package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
)

// Visual areas
// ------------
//
// - The play area: the space the World is aware of.
// - The frame: surrounds the play region.
// - The timer: above the frame.
// - The top bar: above the timer. Contains the menu button and the score.
// - The game area: contains all of the above. Has a fixed size, known at
// compile time.
// - The screen: contains the game area and any margins necessary to fill in
// the application window on the OS. Its size is known only at run time.

const MarginLeft = int64(117)
const MarginRight = int64(118)
const MarginUp = int64(426)
const MarginDown = int64(133)
const GameWidth = PlayAreaWidth + MarginLeft + MarginRight
const GameHeight = PlayAreaHeight + MarginUp + MarginDown

func ButtonRect(x, y, width, height int64) Rectangle {
	return Rectangle{Pt{x, y}, Pt{x + width, y + height}}
}

var homeScreenMenuButton = ButtonRect(38, 38, 137, 137)
var playScreenMenuButton = ButtonRect(467, 1277, 237, 237)
var pausedScreenContinueButton1 = ButtonRect(38, 37, 137, 137)
var pausedScreenContinueButton2 = ButtonRect(303, 807, 137, 137)
var pausedScreenRestartButton = ButtonRect(303, 990, 137, 137)
var pausedScreenHomeButton = ButtonRect(303, 1172, 137, 137)
var gameOverScreenRestartButton = ButtonRect(303, 1175, 137, 137)
var gameOverScreenHomeButton = ButtonRect(303, 1358, 137, 137)

func (g *Gui) Draw(screen *ebiten.Image) {
	// The screen bitmap has the aspect ratio of the application window. We fill
	// it with some background. Then, we select the area inside of screen on
	// which we draw all the actually interesting elements of our game.
	screen.Fill(color.NRGBA{
		R: 180,
		G: 180,
		B: 180,
		A: 255,
	})

	// Draw game area.
	marginX := (int64(screen.Bounds().Size().X) - g.adjustedGameWidth) / 2
	marginY := (int64(screen.Bounds().Size().Y) - g.adjustedGameHeight) / 2

	game := SubImage(screen, NewRectangleI(
		marginX,
		marginY,
		marginX+GameWidth,
		marginY+GameHeight))

	switch g.state {
	case HomeScreen:
		g.DrawHomeScreen(game)
	case PlayScreen:
		g.DrawPlayScreen(game)
	case PausedScreen:
		g.DrawPlayScreen(game)
		g.DrawPausedScreen(game)
	case GameOverScreen:
		g.DrawPlayScreen(game)
		g.DrawGameOverScreen(game)
	case GameWonScreen:
		g.DrawPlayScreen(game)
		g.DrawGameWonScreen(game)
	case Playback:
		g.DrawPlayScreen(game)
	case DebugCrash:
		g.DrawPlayScreen(game)
	default:
		panic("unhandled default case")
	}

	// Draw debug controls.
	if g.adjustedGameHeight > GameHeight {
		debugHorizontal := SubImage(screen, NewRectangleI(
			marginX,
			marginY+GameHeight,
			marginX+g.adjustedGameWidth,
			marginY+g.adjustedGameHeight))
		g.DrawDebugControlsHorizontal(debugHorizontal)
	}

	if g.adjustedGameWidth > GameWidth {
		debugVertical := SubImage(screen, NewRectangleI(
			marginX+GameWidth,
			marginY,
			marginX+g.adjustedGameWidth,
			marginY+GameHeight))
		g.DrawDebugControlsVertical(debugVertical)
	}
}

func (g *Gui) DrawHomeScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgHomeScreen)
}

func (g *Gui) DrawPlayScreen(screen *ebiten.Image) {
	// Highlight the screen area for dev purposes.
	screen.Fill(color.NRGBA{
		R: 230,
		G: 237,
		B: 240,
		A: 255,
	})

	DrawSpriteStretched(screen, g.imgScreenPlay)

	// Draw timer.
	// Draw timer bar going down.
	totalWidth := int64(680)
	timeLeft := totalWidth * g.world.RegularCooldownIdx / g.world.RegularCooldown
	timerBar := SubImage(screen, NewRectangleI(
		270,
		264,
		275+timeLeft,
		290))
	timerBar.Fill(color.NRGBA{
		R: 251,
		G: 150,
		B: 32,
		A: 255,
	})
	DrawSpriteStretched(screen, g.imgTimer)

	// Draw play area.
	playArea := SubImage(screen, NewRectangleI(
		MarginLeft,
		MarginUp,
		GameWidth-MarginRight,
		GameHeight-MarginDown))

	// Draw empty spaces.
	for y := int64(0); y < NRows; y++ {
		for x := int64(0); x < NCols; x++ {
			pos := g.world.CanonicalPosToPixelPos(Pt{x, y})
			DrawSprite(playArea, g.imgBlank, float64(pos.X), float64(pos.Y),
				float64(BrickPixelSize),
				float64(BrickPixelSize))
		}
	}

	// Draw actual bricks.
	// Make sure dragged and falling bricks get drawn on top of canonical ones,
	// so you always see the brick that's moving to be moving on top of the
	// brick that's static.
	// Between dragged and falling I just chose the falling to be on top of the
	// dragged. It will not happen very often, when it does it will be too quick
	// to really notice it, but I feel when it happens it will be because the
	// falling brick will come on top of the dragged brick, which the player
	// is moving around with more hesitation than the falling brick. I am not
	// sure if that makes sense, but between the dragged and the falling brick
	// I just chose for the falling brick to be the dominating one.
	g.DrawBricks(playArea, Canonical)
	g.DrawBricks(playArea, Dragged)
	g.DrawBricks(playArea, Falling)

	// Draw debugging info.
	for _, pt := range g.world.DebugPts {
		DrawPixel(screen, pt, color.NRGBA{
			R: 255,
			G: 0,
			B: 0,
			A: 255,
		})
	}

	// g.DrawText(screen, fmt.Sprintf("ActualTPS: %f", ebiten.ActualTPS()), false,
	// 	false,
	// 	color.NRGBA{
	// 		R: 0,
	// 		G: 100,
	// 		B: 0,
	// 		A: 255,
	// 	})

	if g.state == Playback {
		DrawSprite(screen, g.imgCursor,
			float64(g.mousePt.X),
			float64(g.mousePt.Y),
			50.0, 50.0)
	}
}

func (g *Gui) DrawPausedScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgPausedScreen)
}

func (g *Gui) DrawGameOverScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgGameOverScreen)
}

func (g *Gui) DrawGameWonScreen(screen *ebiten.Image) {

}

func (g *Gui) DrawDebugControlsHorizontal(screen *ebiten.Image) {
	// Background of playback bar.
	screen.Fill(color.NRGBA{
		R: 200,
		G: 200,
		B: 200,
		A: 255,
	})

	// Play/pause button.
	playbarHeight := int64(screen.Bounds().Dy())
	playButtonWidth := playbarHeight
	playButtonHeight := playbarHeight
	playButton := SubImage(screen,
		NewRectangleI(0, 0, playButtonWidth, playButtonHeight))
	if g.playbackPaused {
		DrawSpriteStretched(playButton, g.imgPlaybackPlay)
	} else {
		DrawSpriteStretched(playButton, g.imgPlaybackPause)
	}
	// Remember the region so that Update() can react when it's clicked.
	r := playButton.Bounds()
	g.buttonPlaybackPlay = Rectangle{
		Min: Pt{int64(r.Min.X), int64(r.Min.Y)},
		Max: Pt{int64(r.Max.X), int64(r.Max.Y)},
	}

	// Play bar.
	barXMargin := int64(10)
	barX := playButtonWidth + barXMargin
	barWidth := int64(screen.Bounds().Dx()) - barX - barXMargin
	bar := SubImage(screen,
		NewRectangleI(barX, 0, barX+barWidth, playbarHeight))
	DrawSpriteStretched(bar, g.imgPlayBar)
	// Remember the region so that Update() can react when it's clicked.
	g.buttonPlaybackBar = bar.Bounds()

	// Playback bar cursor.
	cursorWidth := float64(playbarHeight)
	cursorHeight := float64(playbarHeight)
	factor := float64(g.frameIdx) / float64(len(g.playthrough.History))
	cursorX := factor*float64(g.buttonPlaybackBar.Size().X) - cursorWidth/2
	DrawSprite(bar, g.imgPlaybackCursor, cursorX, 0, cursorWidth, cursorHeight)
}

func (g *Gui) DrawDebugControlsVertical(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{
		R: 0,
		G: 0,
		B: 255,
		A: 255,
	})
}

func (g *Gui) DrawBricks(play *ebiten.Image, s BrickState) {
	for _, b := range g.world.Bricks {
		if b.State != s {
			continue
		}
		pos := b.PixelPos
		img := g.imgBrick[b.Val]
		DrawSprite(play, img, float64(pos.X), float64(pos.Y),
			float64(BrickPixelSize),
			float64(BrickPixelSize))
	}
}

func (g *Gui) DrawText(screen *ebiten.Image, message string, centerX bool, centerY bool, color color.Color) {
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
	// I receive the application window's actual width and height, via
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
	// - Have a "game area" that I can reason about easily, no matter the
	// aspect ratio or the resolution of the user's screen.
	// - Have a "game area" that makes sense for a vertical smartphone. So,
	// taller than wider. But don't worry too much about matching the exact
	// aspect ratio of a particular smartphone model.
	// - Have a "game area" with enough pixels that most of the time
	// ebitengine will have to shrink the image, not enlarge it.
	//
	// Solution:
	// - Have a fixed "game area". The actual value is computed from constants,
	// but let's say it is 1200 x 2000 pixels in the end.
	// - Compute screenWidth and screenHeight so that the aspect ratio of the
	// screen bitmap is the same as the aspect ratio of the game window. This
	// means screenWidth / screenHeight = outsideWidth / outsideHeight.
	// - Compute screenWidth and screenHeight such that the game area is as
	// large as it can be, but still fit inside the screen. This means either
	// screenWidth = 1200 or screenHeight = 2000.
	//
	// Extra complication:
	// - For debug purposes I find it very useful to have extra space to place
	// controls.
	// - The mechanism above guarantees a region of GameWidth x GameHeight.
	// - The easiest way to add space for my controls is to add margins to
	// GameWidth and GameHeight.
	// - Simply use g.adjustedGameWidth and g.adjustedGameHeight.

	// Find out if we need to match the screen width to the game width or the
	// screen height to the game height.
	// The aspect ratio of a rectangle is width / height. As an aspect ratio
	// goes down to 0, the rectangle gets progressively thinner/taller.
	// aspectRatio(rectangleA) < aspectRatio(rectangleB) means that rectangleA
	// is thinner/taller than rectangleB.
	// So if I scale rectangleB to the maximum size that fits inside rectangleA
	// then rectangleB will fill the width of rectangleA, and there will be
	// space left at the top and the bottom.
	// So, if aspectRatio(rectangleA) < aspectRatio(rectangleB), I will have
	// rectangleA.width == rectangleB.width.
	// I want game to fit inside screen, so screen is A and game is B.
	outsideAspectRatio := float64(outsideWidth) / float64(outsideHeight)
	screenAspectRatio := outsideAspectRatio
	gameAspectRatio := float64(g.adjustedGameWidth) / float64(g.adjustedGameHeight)
	if screenAspectRatio < gameAspectRatio {
		screenWidth = int(g.adjustedGameWidth)
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenHeight = int(float64(screenWidth) / screenAspectRatio)
	} else {
		screenHeight = int(g.adjustedGameHeight)
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenWidth = int(float64(screenHeight) * screenAspectRatio)
	}

	// Store these values in Gui so that Update() can use them as well,
	// otherwise only Draw() will have access to them via the size of the
	// screen parameter it receives.
	g.screenWidth = int64(screenWidth)
	g.screenHeight = int64(screenHeight)
	g.gameAreaOrigin.X = (g.screenWidth - g.adjustedGameWidth) / 2
	g.gameAreaOrigin.Y = (g.screenHeight - g.adjustedGameHeight) / 2
	g.playAreaOrigin = g.gameAreaOrigin.Plus(Pt{MarginLeft, MarginUp})
	return
}

func (g *Gui) LoadGuiData() {
	// Read from the disk over and over until a full read is possible.
	// This repetition is meant to avoid crashes due to reading files
	// while they are still being written.
	// It's a hack but possibly a quick and very useful one.
	// This repeated reading is only useful when we're not reading from the
	// embedded filesystem. When we're reading from the embedded filesystem we
	// want to crash as soon as possible. We might be in the browser, in which
	// case we want to see an error in the developer console instead of a page
	// that keeps trying to load and reports nothing.
	previousVal := CheckCrashes
	if g.FSys == nil {
		CheckCrashes = false
	}
	for {
		CheckFailed = nil
		g.imgBlank = LoadImage(g.FSys, "data/gui/blank.png")
		for i := int64(1); i <= g.world.MaxBrickValue; i++ {
			filename := fmt.Sprintf("data/gui/%02d.png", i)
			g.imgBrick[i] = LoadImage(g.FSys, filename)
		}
		g.imgCursor = LoadImage(g.FSys, "data/gui/cursor.png")
		g.imgPlaybackCursor = LoadImage(g.FSys, "data/gui/playback-cursor.png")
		g.imgPlaybackPause = LoadImage(g.FSys, "data/gui/playback-pause.png")
		g.imgPlaybackPlay = LoadImage(g.FSys, "data/gui/playback-play.png")
		g.imgPlayBar = LoadImage(g.FSys, "data/gui/playbar.png")
		g.imgTimer = LoadImage(g.FSys, "data/gui/timer.png")
		g.imgHomeScreen = LoadImage(g.FSys, "data/gui/screen-home.png")
		g.imgScreenPlay = LoadImage(g.FSys, "data/gui/screen-play.png")
		g.imgPausedScreen = LoadImage(g.FSys, "data/gui/screen-paused.png")
		g.imgGameOverScreen = LoadImage(g.FSys, "data/gui/screen-game-over.png")

		if CheckFailed == nil {
			break
		}
	}
	CheckCrashes = previousVal

	g.UpdateWindowSize()
}

func (g *Gui) UpdateWindowSize() {
	// ebiten.SetWindowSize(int(g.adjustedGameWidth)/3, int(g.adjustedGameHeight)/3)
	width, height := ebiten.ScreenSizeInFullscreen()
	size := min(width, height) * 8 / 10
	ebiten.SetWindowSize(size, size)
	// ebiten.SetWindowSize(460, 700)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Clone1")
}

func (g *Gui) ScreenToPlayArea(pt Pt) Pt {
	return pt.Minus(g.playAreaOrigin)
}

func (g *Gui) PlayAreaToScreen(pt Pt) Pt {
	return pt.Plus(g.playAreaOrigin)
}
