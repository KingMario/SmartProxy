//go:build !darwin

package main

func bindSocketToInterface(fd uintptr, network string, ifIndex int) {
}
