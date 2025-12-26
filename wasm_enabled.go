//go:build js && wasm

package main

import (
	"syscall/js"
)

func getUsername() string {
	// Retrieve parameter from JavaScript global scope.
	return js.Global().Get("username").String()
}

func WriteFile(name string, data []byte) {
}

func AppendToFile(name string, str string) {
}

func MakeDir(name string) {
}

func DeleteDir(name string) {
}

func ChDir(name string) {
}
