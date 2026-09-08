package main

import (
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsInterfaceObserverInitialNotification(t *testing.T) {
	changed := make(chan struct{}, 2)
	stop, err := observeInterfaceChanges(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("no initial native notification")
	}
	stop()
	stop()
	count := 0
	interfaceObservers.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Fatalf("retained %d callback contexts", count)
	}
}

func TestWindowsInterfaceErrors(t *testing.T) {
	if !isInterfaceError(windows.WSAEADDRNOTAVAIL) {
		t.Fatal("stale Windows source address not recognized")
	}
	if isInterfaceError(windows.WSAECONNREFUSED) {
		t.Fatal("connection refusal should not refresh the interface")
	}
}
