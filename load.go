package main

import (
	"fmt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"io/fs"
)

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
	if _, ok := g.FSys.(fs.FS); ok {
		CheckCrashes = false
	}
	for {
		CheckFailed = nil
		if g.devModeEnabled {
			LoadYAML(g.FSys, "data/config-dev.yaml", &g.Config)
		} else {
			LoadYAML(g.FSys, "data/config.yaml", &g.Config)
		}
		g.imgBlank = LoadImage(g.FSys, "data/gui/blank.png")
		for i := int64(1); i <= 30; i++ {
			filename := fmt.Sprintf("data/gui/%02d.png", i)
			g.imgBrick[i] = LoadImage(g.FSys, filename)
		}
		for i := int64(0); i <= 9; i++ {
			filename := fmt.Sprintf("data/gui/digit%d.png", i)
			g.imgDigit[i] = LoadImage(g.FSys, filename)
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
		g.imgGameWonScreen = LoadImage(g.FSys, "data/gui/screen-game-won.png")
		g.imgChain = LoadImage(g.FSys, "data/gui/chain.png")
		g.animSplashRadial = NewAnimation(g.FSys, "data/gui/splash-radial")
		g.animSplashDown = NewAnimation(g.FSys, "data/gui/splash-down")

		if CheckFailed == nil {
			break
		}
	}
	CheckCrashes = previousVal

	g.visWorld = NewVisWorld(g.Animations)
	g.UpdateWindowSize()

	// Load the Arial font.
	fontData, err := opentype.Parse(goregular.TTF)
	Check(err)

	g.defaultFont, err = opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    44,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
	Check(err)
}
