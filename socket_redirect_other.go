//go:build !windows

package main

import (
	"context"
	"net"
	"time"
)

func dialRedirectedVPN(ctx context.Context, target string, info interfaceInfo, timeout time.Duration, dialErr error) (net.Conn, error) {
	return nil, dialErr
}
