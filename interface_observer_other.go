//go:build (!darwin && !windows) || (darwin && !cgo)

package main

import "fmt"

func observeInterfaceChanges(changed func()) (func(), error) {
	return nil, fmt.Errorf("native interface observer is unavailable on this build")
}
