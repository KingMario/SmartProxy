package main

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

func isPlatformInterfaceError(err error) bool {
	for _, code := range []syscall.Errno{windows.WSAENETDOWN, windows.WSAENETUNREACH, windows.WSAENETRESET, windows.WSAEHOSTUNREACH, windows.WSAEADDRNOTAVAIL} {
		if errors.Is(err, code) {
			return true
		}
	}
	return false
}
