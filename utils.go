package main

import (
	"errors"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

var CheckCrashes = true
var CheckFailed error

func Check(e error) {
	if e != nil {
		CheckFailed = e
		if CheckCrashes {
			panic(e)
		}
	}
}

func LoadImage(fsys FS, str string) *ebiten.Image {
	file, err := fsys.Open(str)
	defer func(file fs.File) { Check(file.Close()) }(file)
	Check(err)

	img, _, err := image.Decode(file)
	Check(err)
	if err != nil {
		return nil
	}

	return ebiten.NewImageFromImage(img)
}

func CloseFile(f fs.File) {
	Check(f.Close())
}

func WriteFile(name string, data []byte) {
	err := os.WriteFile(name, data, 0644)
	Check(err)
}

// CopyFile copies a file (not a folder) from source to destination.
// Apparently copying files has all sorts of edge cases and Go doesn't provide
// a default function in its standard library for this because the developer
// should decide how to handle the edge cases. In my case, I just want a new
// file with the same contents as the old file. If the destination file already
// exists, it is overwritten.
func CopyFile(source, dest string) {
	sourceFileStat, err := os.Stat(source)
	Check(err)

	if !sourceFileStat.Mode().IsRegular() {
		Check(fmt.Errorf("%s is not a regular file", source))
	}

	sourceReader, err := os.Open(source)
	Check(err)
	defer func(file *os.File) { Check(file.Close()) }(sourceReader)

	destWriter, err := os.Create(dest)
	Check(err)
	defer func(file *os.File) { Check(file.Close()) }(destWriter)

	_, err = io.Copy(destWriter, sourceReader)
	Check(err)
}

func DeleteFile(name string) {
	err := os.Remove(name)
	if !errors.Is(err, os.ErrNotExist) {
		Check(err)
	}
}

func ReadFile(name string) []byte {
	data, err := os.ReadFile(name)
	Check(err)
	return data
}

func FileExists(fsys FS, name string) bool {
	file, err := fsys.Open(name)
	if err == nil {
		CloseFile(file)
		return true
	} else {
		return false
	}
}

func AppendToFile(name string, str string) {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	Check(err)
	defer func(file *os.File) { Check(file.Close()) }(f)
	_, err = f.WriteString(str)
	Check(err)
}

func MakeDir(name string) {
	err := os.MkdirAll(name, 0644)
	Check(err)
}

func DeleteDir(name string) {
	err := os.RemoveAll(name)
	Check(err)
}

func ChDir(name string) {
	err := os.Chdir(name)
	Check(err)
}

func GetFiles(fsys FS, dir string, pattern string) []string {
	var files []string
	entries, err := fsys.ReadDir(dir)
	Check(err)
	for _, entry := range entries {
		matched, err := filepath.Match(pattern, entry.Name())
		Check(err)
		if matched {
			files = append(files, dir+"/"+entry.Name())
		}
	}
	return files
}

type FolderWatcher struct {
	Folder string
	times  []time.Time
}

func (f *FolderWatcher) FolderContentsChanged() bool {
	if f.Folder == "" {
		return false
	}

	files, err := os.ReadDir(f.Folder)
	Check(err)
	if len(files) != len(f.times) {
		f.times = make([]time.Time, len(files))
	}
	changed := false
	for idx, file := range files {
		info, err := file.Info()
		Check(err)
		if f.times[idx] != info.ModTime() {
			changed = true
			f.times[idx] = info.ModTime()
		}
	}
	return changed
}
