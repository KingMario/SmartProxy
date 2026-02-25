//go:build windows

package main

import (
	"errors"
	"os"
)

func acquireInstanceLock(lockFile string) (func(), error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}

	released := false
	release := func() {
		if released {
			return
		}
		released = true
		_ = f.Close()
		_ = os.Remove(lockFile)
	}

	return release, nil
}
