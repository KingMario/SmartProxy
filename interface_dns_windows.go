package main

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func interfaceDNSServers(ctx context.Context, index int) ([]string, error) {
	var size uint32 = 16 * 1024
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		buffer := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		err := windows.GetAdaptersAddresses(windows.AF_INET, 0, 0, first, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return nil, err
		}
		for adapter := first; adapter != nil; adapter = adapter.Next {
			if int(adapter.IfIndex) != index {
				continue
			}
			var servers []string
			for dns := adapter.FirstDnsServerAddress; dns != nil; dns = dns.Next {
				if ip := dns.Address.IP(); ip != nil && ip.To4() != nil && !ip.IsUnspecified() {
					servers = append(servers, ip.String())
				}
			}
			return servers, nil
		}
		return nil, fmt.Errorf("interface index %d not found", index)
	}
	return nil, fmt.Errorf("adapter table changed repeatedly")
}
