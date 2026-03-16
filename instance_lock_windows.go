//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

func acquireInstanceLock(lockFile string) (func(), error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// Try to acquire an exclusive lock without blocking
	ol := new(syscall.Overlapped)
	r1, _, _ := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		// Failed to lock — another instance holds it
		_ = f.Close()
		return nil, ErrAlreadyRunning
	}

	released := false
	release := func() {
		if released {
			return
		}
		released = true
		ol := new(syscall.Overlapped)
		procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(ol)))
		_ = f.Close()
		_ = os.Remove(lockFile)
	}

	return release, nil
}
