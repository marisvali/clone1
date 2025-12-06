//go:build !(js && wasm)

package main

func getUsername() string {
	return "vali-dev"
}
