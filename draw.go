package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
)

func (g *Gui) Draw(screen *ebiten.Image) {
	// The screen bitmap has the aspect ratio of the application window. We fill
	// it with some background. Then, we select the area inside of screen on
	// which we draw all the actually interesting elements of our gameScreen.
	screen.Fill(color.NRGBA{
		R: 180,
		G: 180,
		B: 180,
		A: 255,
	})

	// Draw the game area.
	gameScreen := SubImage(screen, g.gameArea)

	switch g.state {
	case HomeScreen:
		g.DrawHomeScreen(gameScreen)
	case PlayScreen:
		g.DrawPlayScreen(gameScreen)
	case PausedScreen:
		g.DrawPlayScreen(gameScreen)
		g.DrawPausedScreen(gameScreen)
	case GameOverScreen:
		g.DrawPlayScreen(gameScreen)
		g.DrawGameOverScreen(gameScreen)
	case GameWonScreen:
		g.DrawPlayScreen(gameScreen)
		g.DrawGameWonScreen(gameScreen)
	case Playback:
		g.DrawPlayScreen(gameScreen)
	case DebugCrash:
		g.DrawPlayScreen(gameScreen)
	default:
		panic("unhandled default case")
	}

	// Draw debug controls.
	if g.enableDebugAreas {
		g.DrawDebugControlsHorizontal(SubImage(screen, g.horizontalDebugArea))
		g.DrawDebugControlsVertical(SubImage(screen, g.verticalDebugArea))
	}
}

func (g *Gui) DrawHomeScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgHomeScreen)
}

func (g *Gui) DrawPlayScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgScreenPlay)

	// Draw time left in orange.
	timeLeftWidth := playScreenTimerArea.Width() *
		g.world.RegularCooldownIdx /
		g.world.RegularCooldown
	timeLeftArea := playScreenTimerArea
	timeLeftArea.Max.X = timeLeftArea.Min.X + timeLeftWidth
	SubImage(screen, timeLeftArea).Fill(color.NRGBA{
		R: 251,
		G: 150,
		B: 32,
		A: 255,
	})
	// Draw timer-only sprite with a transparent area, over the time left to
	// round off the edges.
	DrawSpriteStretched(screen, g.imgTimer)

	// Draw world.
	worldScreen := SubImage(screen, playScreenWorldArea)

	// Draw empty spaces.
	for y := int64(0); y < NRows; y++ {
		for x := int64(0); x < NCols; x++ {
			pos := g.world.CanonicalPosToPixelPos(Pt{x, y})
			DrawSprite(worldScreen, g.imgBlank, float64(pos.X), float64(pos.Y),
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
	g.DrawBricks(worldScreen, Canonical)
	g.DrawBricks(worldScreen, Dragged)
	g.DrawBricks(worldScreen, Falling)

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

	if g.state == Playback || g.state == DebugCrash {
		pos := g.ScreenToGame(g.virtualPointerPos)
		DrawSprite(screen, g.imgCursor,
			float64(pos.X), float64(pos.Y),
			50.0, 50.0)
	}
}

func (g *Gui) DrawPausedScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgPausedScreen)
}

func (g *Gui) DrawGameOverScreen(screen *ebiten.Image) {
	DrawSpriteStretched(screen, g.imgGameOverScreen)
}

func (g *Gui) DrawGameWonScreen(uiScreen *ebiten.Image) {

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
	button := SubImage(screen, debugPlayButton)
	if g.playbackPaused {
		DrawSpriteStretched(button, g.imgPlaybackPlay)
	} else {
		DrawSpriteStretched(button, g.imgPlaybackPause)
	}

	// Play bar.
	bar := SubImage(screen, debugPlayBar)
	DrawSpriteStretched(bar, g.imgPlayBar)

	// Playback bar cursor.
	cursorWidth := float64(debugPlayBar.Height())
	cursorHeight := float64(debugPlayBar.Height())
	factor := float64(g.frameIdx) / float64(len(g.playthrough.History))
	cursorX := factor*float64(debugPlayBar.Width()) - cursorWidth/2
	DrawSprite(bar, g.imgPlaybackCursor, cursorX, 0, cursorWidth, cursorHeight)
}

func (g *Gui) DrawDebugControlsVertical(uiScreen *ebiten.Image) {
	uiScreen.Fill(color.NRGBA{
		R: 0,
		G: 0,
		B: 255,
		A: 255,
	})
}

func (g *Gui) DrawBricks(worldScreen *ebiten.Image, s BrickState) {
	for _, b := range g.world.Bricks {
		if b.State != s {
			continue
		}
		pos := b.PixelPos
		img := g.imgBrick[b.Val]
		DrawSprite(worldScreen, img, float64(pos.X), float64(pos.Y),
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
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Clone1")
}
