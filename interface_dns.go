package main

import (
	"net"
	"strconv"
	"strings"
)

// Use the selected interface's scoped resolver section from scutil --dns.
// Supplemental and service-specific resolvers have separate routing scopes.
func scopedDNSServers(output string, index int) []string {
	_, scoped, ok := strings.Cut(output, "DNS configuration (for scoped queries)")
	if !ok {
		return nil
	}
	scoped, _, _ = strings.Cut(scoped, "DNS configuration (")
	var result []string
	seen := map[string]bool{}
	for _, block := range strings.Split(scoped, "resolver #") {
		matches := false
		var servers []string
		for _, line := range strings.Split(block, "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
			if !ok {
				continue
			}
			fields := strings.Fields(value)
			if len(fields) == 0 {
				continue
			}
			if strings.TrimSpace(key) == "if_index" {
				matches = fields[0] == strconv.Itoa(index)
			}
			if strings.HasPrefix(key, "nameserver[") {
				if ip := net.ParseIP(fields[0]); ip != nil && ip.To4() != nil && !ip.IsUnspecified() {
					servers = append(servers, ip.String())
				}
			}
		}
		if matches {
			for _, server := range servers {
				if !seen[server] {
					result = append(result, server)
					seen[server] = true
				}
			}
		}
	}
	return result
}
