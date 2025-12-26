//go:build !(js && wasm)

package main

import "os"

func getUsername() string {
	return "vali-dev"
}

func WriteFile(name string, data []byte) {
	err := os.WriteFile(name, data, 0644)
	Check(err)
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
