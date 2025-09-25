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

//go:embed data/*
var embeddedFiles embed.FS

const (
	playWidth  = 1200
	playHeight = 2000
)

type Gui struct {
	layout         Pt
	world          World
	FSys           FS
	imgBlank       *ebiten.Image
	img1           *ebiten.Image
	img2           *ebiten.Image
	img3           *ebiten.Image
	imgFalling     *ebiten.Image
	folderWatcher1 FolderWatcher
	defaultFont    font.Face
	screenWidth    int
	screenHeight   int
}

func main() {
	// ebiten.SetWindowSize(900, 900)
	ebiten.SetWindowPosition(50, 100)

	var g Gui
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
	g.world = NewWorld()

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
