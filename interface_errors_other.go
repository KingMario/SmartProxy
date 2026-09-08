//go:build !windows

package main

func isPlatformInterfaceError(err error) bool { return false }
