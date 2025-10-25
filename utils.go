package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
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

func Serialize(w io.Writer, data any) {
	err := binary.Write(w, binary.LittleEndian, data)
	Check(err)
}

func Deserialize(r io.Reader, data any) {
	err := binary.Read(r, binary.LittleEndian, data)
	Check(err)
}

func SerializeSlice[T any](buf *bytes.Buffer, s []T) {
	Serialize(buf, int64(len(s)))
	Serialize(buf, s)
}

func DeserializeSlice[T any](buf *bytes.Buffer, s *[]T) {
	var lenSlice int64
	Deserialize(buf, &lenSlice)
	*s = make([]T, lenSlice)
	Deserialize(buf, *s)
}

func Unzip(data []byte) []byte {
	// Get a bytes.Reader, which implements the io.ReaderAt interface required
	// by the zip.NewReader() function.
	bytesReader := bytes.NewReader(data)

	// Open a zip archive for reading.
	r, err := zip.NewReader(bytesReader, int64(len(data)))
	Check(err)

	// We assume there's exactly 1 file in the zip archive.
	if len(r.File) != 1 {
		Check(errors.New(fmt.Sprintf("expected exactly one file in zip archive, got: %d", len(r.File))))
	}

	// Get a reader for that 1 file.
	f := r.File[0]
	rc, err := f.Open()
	Check(err)
	defer func(rc io.ReadCloser) { Check(rc.Close()) }(rc)

	// Keep reading bytes, 1024 bytes at a time.
	buffer := make([]byte, 1024)
	fullContent := make([]byte, 0, 1024)
	for {
		nbytesActuallyRead, err := rc.Read(buffer)
		fullContent = append(fullContent, buffer[:nbytesActuallyRead]...)
		if err == io.EOF {
			break
		}
		Check(err)
		if nbytesActuallyRead == 0 {
			break
		}
	}

	// Return bytes.
	return fullContent
}

func UnzipFromFile(filename string) []byte {
	return Unzip(ReadFile(filename))
}

func Zip(data []byte) []byte {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Create a single file inside it called "recorded-inputs".
	f, err := w.Create("recorded-inputs")
	Check(err)

	// Write/compress the data to the file inside the zip.
	_, err = f.Write(data)
	Check(err)

	// Make sure to check the error on Close.
	err = w.Close()
	Check(err)

	return buf.Bytes()
}

func ZipToFile(filename string, data []byte) {
	// Actually write the zip to disk.
	WriteFile(filename, Zip(data))
}

func Sqr(x int64) int64 {
	return x * x
}

func Remove[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
