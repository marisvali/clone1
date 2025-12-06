//go:build js && wasm

package main

import (
	"syscall/js"
)

func getUsername() string {
	// Retrieve parameter from JavaScript global scope.
	return js.Global().Get("username").String()
}
