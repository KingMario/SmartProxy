package main

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

func dialInterfaceTCP(ctx context.Context, target string, info interfaceInfo, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := newInterfaceDialer(info, timeout).DialContext(ctx, "tcp", target)
	if err == nil {
		return conn, nil
	}
	return dialRedirectedVPN(ctx, target, info, timeout, err)
}

// newInterfaceDialer fails the connection if either raw socket access or binding
// fails. Never let an interface-specific connection fall back to system routing.
func newInterfaceDialer(info interfaceInfo, timeout time.Duration) *net.Dialer {
	dialer := &net.Dialer{Timeout: timeout}
	if info.ip != "" {
		dialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(info.ip)}
	}
	if info.index != 0 {
		dialer.Control = interfaceSocketControl(info.index)
	}
	return dialer
}

func interfaceSocketControl(index int) func(string, string, syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var bindErr error
		if err := c.Control(func(fd uintptr) {
			bindErr = bindSocketToInterface(fd, network, index)
		}); err != nil {
			return fmt.Errorf("access socket for interface %d: %w", index, err)
		}
		if bindErr != nil {
			return fmt.Errorf("bind %s socket to interface %d: %w", network, index, bindErr)
		}
		return nil
	}
}
