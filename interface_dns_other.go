//go:build !windows && !darwin

package main

import (
	"context"
	"fmt"
)

func interfaceDNSServers(ctx context.Context, index int) ([]string, error) {
	return nil, fmt.Errorf("automatic interface DNS discovery is unavailable; enter DNS server IPs")
}
