//go:build darwin

package main

import (
	"strings"
	"syscall"
)

const (
	IP_BOUND_IF   = 0x19
	IPV6_BOUND_IF = 0x7D
)

func bindSocketToInterface(fd uintptr, network string, ifIndex int) {
	if strings.Contains(network, "tcp6") {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, IPV6_BOUND_IF, ifIndex)
	} else {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, IP_BOUND_IF, ifIndex)
	}
}
