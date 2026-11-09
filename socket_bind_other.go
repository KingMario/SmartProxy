//go:build !darwin && !windows

package main

func bindSocketToInterface(fd uintptr, network string, ifIndex int) error {
	return nil
}
