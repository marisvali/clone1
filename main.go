package main

import (
	"embed"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
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

type GameState int64

const (
	HomeScreen GameState = iota
	PlayScreen
	PausedScreen
	GameOverScreen
	GameWonScreen
	Playback
	DebugCrash
)

type Gui struct {
	layout              Pt
	world               World
	FSys                FS
	imgBlank            *ebiten.Image
	imgBrick            [31]*ebiten.Image
	imgDigit            [10]*ebiten.Image
	imgFalling          *ebiten.Image
	imgCursor           *ebiten.Image
	imgPlaybackCursor   *ebiten.Image
	imgPlaybackPause    *ebiten.Image
	imgPlaybackPlay     *ebiten.Image
	imgPlayBar          *ebiten.Image
	imgFrame            *ebiten.Image
	imgTimer            *ebiten.Image
	imgTopbar           *ebiten.Image
	imgHomeScreen       *ebiten.Image
	imgScreenPlay       *ebiten.Image
	imgPausedScreen     *ebiten.Image
	imgGameOverScreen   *ebiten.Image
	folderWatcher1      FolderWatcher
	defaultFont         font.Face
	playthrough         Playthrough
	recordingFile       string
	frameIdx            int64
	state               GameState
	virtualPointerPos   Pt
	debugMarginWidth    int64
	debugMarginHeight   int64
	playbackPaused      bool
	pointer             PointerState
	pressedKeys         []ebiten.Key
	justPressedKeys     []ebiten.Key // keys pressed in this frame
	FrameSkipAltArrow   int64
	FrameSkipShiftArrow int64
	FrameSkipArrow      int64
	enableDebugAreas    bool
	slowdownFactor      int64       // 1 - does nothing, 2 - game is twice as slow etc
	accumulatedInput    PlayerInput // only relevant for slowdownFactor > 1, see
	// the implementation for a more detailed explanation
	gameArea            Rectangle
	horizontalDebugArea Rectangle
	verticalDebugArea   Rectangle
	bestScore           int64
}

type PointerState struct {
	Pressed      bool
	JustPressed  bool
	JustReleased bool
	Pos          Pt
}

func main() {
	// ebiten.SetWindowSize(900, 900)
	ebiten.SetWindowPosition(1000, 100)

	var g Gui
	g.playthrough.InputVersion = InputVersion
	g.playthrough.SimulationVersion = SimulationVersion
	g.playthrough.ReleaseVersion = ReleaseVersion
	g.recordingFile = "last-recording.clone1"
	g.FrameSkipAltArrow = 1
	g.FrameSkipShiftArrow = 10
	g.FrameSkipArrow = 1
	g.slowdownFactor = 1
	g.state = PlayScreen
	// g.state = DebugCrash
	g.state = HomeScreen
	// g.state = Playback

	if len(os.Args) == 2 {
		g.recordingFile = os.Args[1]
		g.state = Playback
	}

	if g.state == Playback || g.state == DebugCrash {
		g.playthrough = DeserializePlaythrough(ReadFile(g.recordingFile))
		g.enableDebugAreas = true
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

	g.LoadGuiData()

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
