package main

import (
	"embed"
	"fmt"
	"github.com/google/uuid"
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
	imgBrickFrame       *ebiten.Image
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
	uploadDataChannel     chan uploadData
	panicHappened         bool
	panicMsg              string
}

type uploadData struct {
	user              string
	releaseVersion    int64
	simulationVersion int64
	inputVersion      int64
	playthrough       *Playthrough
}

type Config struct {
	SlowdownFactor        int64  `yaml:"SlowdownFactor"`
	StartState            string `yaml:"StartState"`
	PlaybackFile          string `yaml:"PlaybackFile"`
	RecordToFile          bool   `yaml:"RecordToFile"`
	RecordToFileOnError   bool   `yaml:"RecordToFileOnError"`
	RecordingFile         string `yaml:"RecordingFile"`
	LoadTest              bool   `yaml:"LoadTest"`
	TestFile              string `yaml:"TestFile"`
	AllowOverlappingDrags bool   `yaml:"AllowOverlappingDrags"`
	DisplayFPS            bool   `yaml:"DisplayFPS"`
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
	var g Gui
	defer g.HandlePanic()
	// ebiten.SetWindowSize(900, 900)
	ebiten.SetWindowPosition(1000, 100)

	g.playthrough.InputVersion = InputVersion
	g.playthrough.SimulationVersion = SimulationVersion
	g.playthrough.ReleaseVersion = ReleaseVersion

	g.username = getUsername()
	// A channel size of 10 means the channel will buffer 10 inputs before it is
	// full and it blocks. Hopefully, when uploading data, a size of 10 is
	// sufficient.
	g.uploadDataChannel = make(chan uploadData, 10)
	go g.UploadPlaythroughs(g.uploadDataChannel)
	g.UserData = LoadUserData(g.username)
	// A channel size of 10 means the channel will buffer 10 inputs before it is
	// full and it blocks. Hopefully, when uploading data, a size of 10 is
	// sufficient.
	g.uploadUserDataChannel = make(chan UserData, 10)
	go g.UploadUserData(g.username, g.uploadUserDataChannel)
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
		g.world = NewWorldFromPlaythrough(g.playthrough)
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
		g.world = NewWorldFromPlaythrough(g.playthrough)
	} else if g.StartState == "Play" {
		g.state = PlayScreen
		if g.LoadTest {
			var test Test
			LoadYAML(g.FSys, g.TestFile, &test)
			g.playthrough.Level = test.GetLevel()
		}
		g.InitializeWorldToNewGame()
	} else {
		panic(fmt.Errorf("invalid g.StartState: %s", g.StartState))
	}
	g.playthrough.AllowOverlappingDrags = g.AllowOverlappingDrags

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

func (g *Gui) InitializeWorldToNewGame() {
	g.playthrough.Id = uuid.New()
	g.playthrough.Seed = time.Now().UnixNano()
	g.playthrough.History = g.playthrough.History[:0]
	g.playthrough.AllowOverlappingDrags = g.AllowOverlappingDrags
	for i := 1; i < 3; i++ {
		// This might fail, but we really do not care that much. The game should
		// not be interrupted by this function failing. If it does fail, just
		// try a couple more times, then give up.
		err := InitializeIdInDbHttp(g.username,
			g.playthrough.ReleaseVersion,
			g.playthrough.SimulationVersion,
			g.playthrough.InputVersion,
			g.playthrough.Id)
		if err == nil {
			break
		}
	}
	g.world = NewWorldFromPlaythrough(g.playthrough)
}

func (g *Gui) HandlePanic() {
	r := recover()
	if r == nil {
		// No panic, nothing to do.
		return
	}
	errorMsg := StackTrace(r)

	// Write to files first, as this should be more reliable than http.
	if g.RecordToFileOnError {
		WriteFile(g.RecordingFile, g.playthrough.Serialize())
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logMessage := fmt.Sprintf(
			"----------------------------------------\n%s %s",
			timestamp, errorMsg)
		AppendToFile("clone1.log", logMessage)
		timestamp = time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("error-%s.clone1", timestamp)
		idx := 1
		for {
			if !FileExists(g.FSys, filename) {
				break
			}
			idx++
			filename = fmt.Sprintf("error-%s-%02d.clone1", timestamp, idx)
		}
		WriteFile(filename, g.playthrough.Serialize())
	}

	// Log the error via HTTP (this is the only thing that will have any effect
	// for errors that happen in the browser, from WASM).
	// Ignore errors, because if this fails and we are in WASM there is nothing
	// more we can do anyway to handle the error.
	_ = LogErrorHttp(
		g.username,
		g.playthrough.ReleaseVersion,
		g.playthrough.SimulationVersion,
		g.playthrough.InputVersion,
		g.playthrough.Id,
		errorMsg,
		g.playthrough.Serialize())

	// Resume panic, we have no recovery solutions.
	// panic(r)
	// TODO: decide best course of action here, panic or display error to user
	g.panicHappened = true
	g.panicMsg = errorMsg[:min(len(errorMsg), 1300)]
}

func (g *Gui) uploadCurrentWorld() {
	// Pass a clone to the channel and not a serialized playthrough.
	// Serialization takes much longer than cloning.
	// Also, it is important for the channel to have some buffer, otherwise this
	// call will block until the previous world instance was uploaded.
	// If the connection to the server drops for a few seconds, either due to
	// the player's connection or the server not being available, it will
	// interrupt the gameplay.
	g.uploadDataChannel <- uploadData{
		g.username,
		g.playthrough.ReleaseVersion,
		g.playthrough.SimulationVersion,
		g.playthrough.InputVersion,
		g.playthrough.Clone()}
}

func (g *Gui) UploadPlaythroughs(ch chan uploadData) {
	defer g.HandlePanic()

	for {
		// Receive a playthrough from the channel.
		// Blocks until a playthrough is received.
		data := <-ch

		// Upload the data.
		// This might fail, but we really do not care that much. The game should
		// not be interrupted by this function failing. If it does fail, just
		// try a couple more times, then give up.
		for i := 1; i < 3; i++ {
			err := UploadDataToDbHttp(data.user,
				data.releaseVersion,
				data.simulationVersion,
				data.inputVersion,
				data.playthrough.Id,
				data.playthrough.Serialize())
			if err == nil {
				break
			}
		}
	}
}
