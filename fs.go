package main

import "io/fs"

// FS groups together the filesystem interfaces that are common between
// embed.FS and what os.DirFS() returns. This way the code that reads data from
// disk can use a FS object and thus work the same if the files are embedded
// or not.
type FS interface {
	fs.FS
	fs.ReadFileFS
	fs.ReadDirFS
}
