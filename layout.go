package main

// Visual areas
// ------------
//
// - The play area: the space the World is aware of. Has a fixed size, known at
// compile time.
// - The game area: the space on which all interesting UI elements are drawn.
// Contains the play area. Has a fixed size, known at compile time.
// - The debug areas: two areas connected to the game area, one at the bottom
// and one on the right. These have a fixed size known at compile time but the
// decision to display them or not happens at runtime.
// - The screen: contains the game area, the debug areas if they are displayed
// and any margins necessary to fill in the application window on the OS. Its
// size is known only at run time.

const PlayMarginLeft = int64(117)
const PlayMarginRight = int64(118)
const PlayMarginUp = int64(426)
const PlayMarginDown = int64(133)
const GameWidth = PlayAreaWidth + PlayMarginLeft + PlayMarginRight
const GameHeight = PlayAreaHeight + PlayMarginUp + PlayMarginDown
const DebugWidth = 0
const DebugHeight = 100

// The areas below are all relative to the game area and known at compile time.
var homeScreenMenuButton = NewRectangleI(38, 38, 137, 137)
var playScreenMenuButton = NewRectangleI(467, 1277, 237, 237)
var playScreenTimerArea = NewRectangleI(270, 264, 690, 20)
var playScreenWorldArea = NewRectangleI(
	PlayMarginLeft,
	PlayMarginUp,
	PlayAreaWidth,
	PlayAreaHeight)
var pausedScreenContinueButton1 = NewRectangleI(38, 37, 137, 137)
var pausedScreenContinueButton2 = NewRectangleI(303, 807, 137, 137)
var pausedScreenRestartButton = NewRectangleI(303, 990, 137, 137)
var pausedScreenHomeButton = NewRectangleI(303, 1172, 137, 137)
var gameOverScreenRestartButton = NewRectangleI(303, 1114, 137, 137)
var gameOverScreenHomeButton = NewRectangleI(303, 1296, 137, 137)
var gameWonScreenRestartButton = NewRectangleI(332, 1236, 137, 137)
var gameWonScreenHomeButton = NewRectangleI(699, 1236, 137, 137)

// The areas below are relative to a debug area and are known at compile time.
var debugPlayButton = NewRectangleI(0, 0, DebugHeight, DebugHeight)
var debugPlayBar = NewRectangleI(DebugHeight+10, 0, GameWidth-DebugHeight-20, DebugHeight)

// Item sizes are set here as it is a matter of layout.
const SplashAnimationSize = 173
const ChainWidth = int64(43)
const ChainHeight = int64(135)

func (g *Gui) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	defer g.HandlePanic()

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
	// - Simply add the debug areas to GameWidth and GameHeight, if they are
	// enabled.

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
	gameWidth := GameWidth
	gameHeight := GameHeight
	if g.enableDebugAreas {
		gameWidth += DebugWidth
		gameHeight += DebugHeight
	}
	gameAspectRatio := float64(gameWidth) / float64(gameHeight)
	if screenAspectRatio < gameAspectRatio {
		screenWidth = int(gameWidth)
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenHeight = int(float64(screenWidth) / screenAspectRatio)
	} else {
		screenHeight = int(gameHeight)
		// screenAspectRatio = screenWidth / screenHeight, which means:
		screenWidth = int(float64(screenHeight) * screenAspectRatio)
	}

	// Define the game area relative to the total screen area.
	g.gameArea.Min.X = (int64(screenWidth) - gameWidth) / 2
	g.gameArea.Min.Y = 0
	g.gameArea.Max.X = g.gameArea.Min.X + GameWidth
	g.gameArea.Max.Y = g.gameArea.Min.Y + GameHeight

	// Define the debug areas relative to the total screen area.
	g.horizontalDebugArea = NewRectangleI(
		g.gameArea.Min.X,
		GameHeight,
		gameWidth,
		DebugHeight)

	g.verticalDebugArea = NewRectangleI(
		GameWidth,
		g.gameArea.Min.Y,
		DebugWidth,
		GameHeight)
	return
}

func (g *Gui) ScreenToGame(pt Pt) Pt {
	return pt.Minus(g.gameArea.Min)
}

func (g *Gui) ScreenToWorld(pt Pt) Pt {
	return pt.Minus(g.gameArea.Min).Minus(playScreenWorldArea.Min)
}

func (g *Gui) WorldToScreen(pt Pt) Pt {
	return pt.Plus(g.gameArea.Min).Plus(playScreenWorldArea.Min)
}

func (g *Gui) ScreenToBottomDebug(pt Pt) Pt {
	return pt.Minus(g.gameArea.Min)
}
