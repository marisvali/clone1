//go:build assert_enabled

package main

func Assert(condition bool) {
	if !condition {
		panic("assert failed")
	}
}
