package main

import (
	"embed"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"image"
	_ "image/png"
	"os"
)

// ReleaseVersion is the version of an executable built and given to someone
// to play, either as a Windows executable or a .wasm on the browser. It is
// meant as a unique label for the functionality that a user/player is presented
// with.
// ReleaseVersion is expected to change very often. Certainly every time a new
// executable is built and sent to someone, it should be tagged with a unique
// ReleaseVersion.
// ReleaseVersion must change when SimulationVersion or InputVersion change.
// But there are many reasons for ReleaseVersion to change when the simulation
// stays the same and the input format stays the same:
// - the executable uses randomly generated levels or fixed levels
// - different fixed levels are included in the executable
// - communication with the server is enabled or disabled
// - graphics change
// All of these changes can be handled by having a generic code that is compiled
// once and depends on a configuration. I very intentionally do not do this.
// The philosophy of this project is currently to release a different executable
// for each variation. The reasons for this:
// - Only the developer is truly comfortable editing the configuration file. If
// the developer has to intervene, he might as well compile a version for the
// user. If it's hard or annoying to compile and release a new version, then
// that process should be improved instead of avoided.
// - I want to keep track of things, who got what experience. The point of a
// configuration is to be able to change things quickly and on the fly. But if
// I want to keep track of things, I must remember to change the release version
// every time I edit the configuration. So the ability to change configurations
// easily is a liability more than a help. It's more helpful, for tracking
// things, to have an unmodifiable binary for each variation.
// - Currently the executables are small enough and I need few enough variations
// that I can easily afford to generate an entire game release for each
// variation (35mb for a Windows .exe and 25mb for a .wasm).
const ReleaseVersion = 999

//go:embed data/*
var embeddedFiles embed.FS

const (
	playWidth  = int64(1200)
	playHeight = int64(2000)
)

type GameState int64

const (
	GameOngoing GameState = iota
	GamePaused
	GameWon
	GameLost
	Playback
	DebugCrash
)

type Gui struct {
	layout              Pt
	world               World
	FSys                FS
	imgBlank            *ebiten.Image
	imgBrick            [31]*ebiten.Image
	imgFalling          *ebiten.Image
	imgCursor           *ebiten.Image
	imgPlaybackCursor   *ebiten.Image
	imgPlaybackPause    *ebiten.Image
	imgPlaybackPlay     *ebiten.Image
	imgPlayBar          *ebiten.Image
	folderWatcher1      FolderWatcher
	defaultFont         font.Face
	screenWidth         int64
	screenHeight        int64
	playthrough         Playthrough
	recordingFile       string
	frameIdx            int64
	state               GameState
	mousePt             Pt // mouse position in this frame
	debugMarginWidth    int64
	debugMarginHeight   int64
	playbackPaused      bool
	buttonPlaybackPlay  image.Rectangle
	buttonPlaybackBar   image.Rectangle
	pressedKeys         []ebiten.Key
	justPressedKeys     []ebiten.Key // keys pressed in this frame
	FrameSkipAltArrow   int64
	FrameSkipShiftArrow int64
	FrameSkipArrow      int64
	adjustedPlayWidth   int64
	adjustedPlayHeight  int64
	slowdownFactor      int64       // 1 - does nothing, 2 - game is twice as slow etc
	accumulatedInput    PlayerInput // only relevant for slowdownFactor > 1, see
	// the implementation for a more detailed explanation
}

func main() {
	// ebiten.SetWindowSize(900, 900)
	ebiten.SetWindowPosition(1000, 100)

	var g Gui
	g.playthrough.InputVersion = InputVersion
	g.playthrough.SimulationVersion = SimulationVersion
	g.playthrough.ReleaseVersion = ReleaseVersion
	g.debugMarginWidth = 0
	g.debugMarginHeight = 100
	g.recordingFile = "last-recording.clone1"
	g.adjustedPlayWidth = playWidth
	g.adjustedPlayHeight = playHeight
	g.FrameSkipAltArrow = 1
	g.FrameSkipShiftArrow = 10
	g.FrameSkipArrow = 1
	g.slowdownFactor = 1
	g.state = GameOngoing
	// g.state = DebugCrash

	if len(os.Args) == 2 {
		g.recordingFile = os.Args[1]
		g.state = Playback
	}

	if g.state == Playback || g.state == DebugCrash {
		g.playthrough = DeserializePlaythrough(ReadFile(g.recordingFile))
		g.adjustedPlayWidth += g.debugMarginWidth
		g.adjustedPlayHeight += g.debugMarginHeight
	}

	if g.state == DebugCrash {
		// Don't crash when we are debugging the crash. This is useful if the
		// crash was caused by one of my asserts:
		// - world.Step() crashed during the last frame, because my assert
		// Check(fmt.Errorf(..))
		// - Now Check() doesn't crash anymore.
		// - I can have the world.Step() with the bug execute, and I can see the
		// results visually
		CheckCrashes = false
	}

	g.world = NewWorldFromPlaythrough(g.playthrough)

	// The last input caused the crash, so run the whole playthrough except the
	// last input. This gives me a chance to see the current state of the world
	// visually, maybe place a breakpoint and inspect the state of the world
	// in the debugger, and then when I'm ready, trigger the bug.
	if g.state == DebugCrash {
		g.frameIdx = int64(len(g.playthrough.History)) - 1
		for i := range g.frameIdx {
			g.world.Step(g.playthrough.History[i])
		}
	}

	if !FileExists(os.DirFS(".").(FS), "data") {
		g.FSys = &embeddedFiles
	} else {
		g.FSys = os.DirFS(".").(FS)
		g.folderWatcher1.Folder = "data/gui"
		// Initialize watchers.
		// Check if folder contents changed but do nothing with the result
		// because we just want the watchers to initialize their internal
		// structures with the current timestamps of files.
		// This is necessary if we want to avoid creating a new world
		// immediately after the first world is created, every time.
		// I want to avoid creating a new world for now because it changes the
		// id of the current world and it messes up the upload of the world
		// to the database.
		g.folderWatcher1.FolderContentsChanged()
	}

	g.loadGuiData()

	// Load the Arial font.
	var err error
	fontData, err := opentype.Parse(goregular.TTF)
	Check(err)

	g.defaultFont, err = opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    44,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
	Check(err)

	err = ebiten.RunGame(&g)
	Check(err)
}
