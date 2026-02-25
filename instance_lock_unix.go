//go:build !windows

package main

import (
	"errors"
	"os"
	"syscall"
)

func acquireInstanceLock(lockFile string) (func(), error) {
	lockFd, err := syscall.Open(lockFile, syscall.O_CREAT|syscall.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(lockFd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = syscall.Close(lockFd)
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
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
		_ = syscall.Close(lockFd)
		_ = os.Remove(lockFile)
	}

	return release, nil
}
