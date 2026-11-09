package main

import (
	"context"
	"fmt"
	"os/exec"
)

func interfaceDNSServers(ctx context.Context, index int) ([]string, error) {
	out, err := exec.CommandContext(ctx, "scutil", "--dns").Output()
	if err != nil {
		return nil, err
	}
	servers := scopedDNSServers(string(out), index)
	if len(servers) == 0 {
		return nil, fmt.Errorf("no scoped IPv4 DNS servers for interface %d; enter DNS server IPs", index)
	}
	return servers, nil
}
