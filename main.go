package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	_ "image/png"
	"os"
	"time"
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
// - communication with the server is enabled or disabled
// - asserts are enabled or disabled
// - writing to the disk is enabled or disabled
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
	Config
	UserData
	Animations
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
	imgGameWonScreen    *ebiten.Image
	imgChainH           *ebiten.Image
	imgChainV           *ebiten.Image
	folderWatcher1      FolderWatcher
	folderWatcher2      FolderWatcher
	defaultFont         font.Face
	playthrough         Playthrough
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
	accumulatedInput    PlayerInput // only relevant for SlowdownFactor > 1, see
	// the implementation for a more detailed explanation
	gameArea              Rectangle
	horizontalDebugArea   Rectangle
	verticalDebugArea     Rectangle
	username              string
	uploadUserDataChannel chan UserData
	visWorld              VisWorld
	devModeEnabled        bool
}

type Config struct {
	SlowdownFactor        int64  `yaml:"SlowdownFactor"`
	StartState            string `yaml:"StartState"`
	PlaybackFile          string `yaml:"PlaybackFile"`
	RecordToFile          bool   `yaml:"RecordToFile"`
	RecordingFile         string `yaml:"RecordingFile"`
	LoadTest              bool   `yaml:"LoadTest"`
	TestFile              string `yaml:"TestFile"`
	AllowOverlappingDrags bool   `yaml:"AllowOverlappingDrags"`
}

type UserData struct {
	BestScore int64 `yaml:"BestScore"`
}

type PointerState struct {
	Pressed      bool
	JustPressed  bool
	JustReleased bool
	Pos          Pt
}

type Animations struct {
	animSplashRadial Animation
	animSplashDown   Animation
}

func main() {
	// ebiten.SetWindowSize(900, 900)
	ebiten.SetWindowPosition(1000, 100)

	var g Gui
	g.playthrough.InputVersion = InputVersion
	g.playthrough.SimulationVersion = SimulationVersion
	g.playthrough.ReleaseVersion = ReleaseVersion
	g.username = getUsername()
	g.UserData = LoadUserData(g.username)
	// A channel size of 10 means the channel will buffer 10 inputs before it is
	// full and it blocks. Hopefully, when uploading data, a size of 10 is
	// sufficient.
	g.uploadUserDataChannel = make(chan UserData, 10)
	go UploadUserData(g.username, g.uploadUserDataChannel)
	g.FrameSkipAltArrow = 1
	g.FrameSkipShiftArrow = 10
	g.FrameSkipArrow = 1

	if !FileExists(os.DirFS(".").(FS), "data") {
		g.FSys = &embeddedFiles
	} else {
		g.FSys = os.DirFS(".").(FS)
		g.folderWatcher1.Folder = "data/gui"
		g.folderWatcher2.Folder = "data"
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
		g.folderWatcher2.FolderContentsChanged()
	}

	filePassedForPlayback := false
	if len(os.Args) == 2 {
		if os.Args[1] == "developer-mode-enabled" {
			g.devModeEnabled = true
		} else {
			filePassedForPlayback = true
		}
	}

	g.LoadGuiData()

	if filePassedForPlayback {
		g.StartState = "Playback"
		g.PlaybackFile = os.Args[1]
	}

	if g.StartState == "Playback" || filePassedForPlayback {
		g.state = Playback
		g.enableDebugAreas = true
		g.playthrough = DeserializePlaythrough(ReadFile(g.PlaybackFile))
	} else if g.StartState == "DebugCrash" {
		g.state = DebugCrash
		g.enableDebugAreas = true
		// Don't crash when we are debugging the crash. This is useful if the
		// crash was caused by one of my asserts:
		// - world.Step() crashed during the last frame, because my assert
		// Check(fmt.Errorf(..))
		// - Now Check() doesn't crash anymore.
		// - I can have the world.Step() with the bug execute, and I can see the
		// results visually
		CheckCrashes = false
		g.playthrough = DeserializePlaythrough(ReadFile(g.PlaybackFile))
	} else if g.StartState == "Play" {
		g.state = PlayScreen
		if g.LoadTest {
			var test Test
			LoadYAML(g.FSys, g.TestFile, &test)
			g.playthrough.Level = test.GetLevel()
		}
		g.playthrough.Seed = time.Now().UnixNano()
	} else {
		panic(fmt.Errorf("invalid g.StartState: %s", g.StartState))
	}

	g.playthrough.AllowOverlappingDrags = g.AllowOverlappingDrags
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

	err := ebiten.RunGame(&g)
	Check(err)
}
