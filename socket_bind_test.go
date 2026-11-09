package main

import (
	"errors"
	"strings"
	"testing"
)

type failedControlConn struct{ err error }

func (c failedControlConn) Control(func(uintptr)) error    { return c.err }
func (c failedControlConn) Read(func(uintptr) bool) error  { panic("unexpected Read") }
func (c failedControlConn) Write(func(uintptr) bool) error { panic("unexpected Write") }

func TestInterfaceControlPropagatesRawConnError(t *testing.T) {
	want := errors.New("raw socket unavailable")
	err := interfaceSocketControl(9)("tcp4", "192.0.2.1:80", failedControlConn{want})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "interface 9") {
		t.Fatalf("got %v, want wrapped raw socket error with interface index", err)
	}
}
