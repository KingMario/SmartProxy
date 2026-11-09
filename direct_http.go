package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Fixed-interface traffic never uses domain routing, DoH, an upstream proxy,
// or the VPN redirect compatibility path. DNS and TCP use the same adapter.
func (p *ProxyServer) dialDirectHTTP(host, target string) (net.Conn, error) {
	p.mu.RLock()
	iface, verbose := p.Config.HTTPProxyIface, p.Config.VerboseLog
	servers := append([]string(nil), p.Config.HTTPProxyDNS...)
	p.mu.RUnlock()
	if iface == "" {
		return nil, fmt.Errorf("select an outgoing interface for HTTP port 17891")
	}
	info, err := p.resolveInterface(iface, true)
	if err != nil {
		return nil, fmt.Errorf("HTTP 17891 interface %q: %w", iface, err)
	}
	if info.index <= 0 || net.ParseIP(info.ip).To4() == nil {
		return nil, fmt.Errorf("HTTP 17891 interface %q has no IPv4 address", iface)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if net.ParseIP(host) == nil && len(servers) == 0 {
		servers, err = interfaceDNSServers(ctx, info.index)
		if err != nil {
			return nil, fmt.Errorf("DNS for interface %q: %w", iface, err)
		}
	}
	endpoints := make([]string, 0, len(servers))
	for _, server := range servers {
		endpoints = append(endpoints, net.JoinHostPort(strings.TrimSpace(server), "53"))
	}
	if verbose {
		p.addLog(fmt.Sprintf("[HTTP 17891] resolving %s via %s (%s), DNS servers %v", host, iface, info.ip, servers))
	}
	ips, err := lookupInterfaceDNS(ctx, host, info, endpoints)
	if err != nil {
		return nil, fmt.Errorf("DNS %s on interface %q: %w", host, iface, err)
	}
	if verbose {
		p.addLog(fmt.Sprintf("[HTTP 17891] %s -> Interface %s; DNS servers %v; addresses %v", target, iface, servers, ips))
	}
	_, port, err := net.SplitHostPort(target)
	if err != nil {
		return nil, err
	}
	var failures []error
	for i, ip := range ips {
		deadline, _ := ctx.Deadline()
		attempt, stop := context.WithTimeout(ctx, time.Until(deadline)/time.Duration(len(ips)-i))
		conn, err := newInterfaceDialer(info, 15*time.Second).DialContext(attempt, "tcp4", net.JoinHostPort(ip, port))
		stop()
		if err == nil {
			if verbose {
				p.addLog(fmt.Sprintf("[HTTP 17891] %s connected via %s (local %s)", target, iface, conn.LocalAddr()))
			}
			return conn, nil
		}
		failures = append(failures, err)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("HTTP 17891 connect %s on %q: %w", target, iface, errors.Join(failures...))
}

func lookupInterfaceDNS(ctx context.Context, host string, info interfaceInfo, servers []string) ([]string, error) {
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil {
			return nil, fmt.Errorf("selected interface requires an IPv4 destination")
		}
		return []string{ip.String()}, nil
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no IPv4 DNS servers for selected interface; configure DNS server IPs")
	}
	var failures []error
	for _, server := range servers {
		// TCP avoids VPN-intercepted UDP DNS and its timeout on multi-VPN hosts.
		// If TCP is unavailable, retry UDP against the same server and adapter.
		for _, forceTCP := range []bool{true, false} {
			resolver := &net.Resolver{PreferGo: true, StrictErrors: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				if forceTCP {
					network = "tcp"
				}
				dialer := newInterfaceDialer(info, 3*time.Second)
				if strings.HasPrefix(network, "udp") {
					dialer.LocalAddr = &net.UDPAddr{IP: net.ParseIP(info.ip)}
				}
				family := network
				if !strings.HasSuffix(family, "4") && !strings.HasSuffix(family, "6") {
					family += "4"
				}
				return dialer.DialContext(ctx, family, server)
			}}
			attempt, cancel := context.WithTimeout(ctx, 3*time.Second)
			ips, err := resolver.LookupIP(attempt, "ip4", host)
			cancel()
			if err == nil && len(ips) > 0 {
				result := make([]string, len(ips))
				for i, ip := range ips {
					result[i] = ip.String()
				}
				return result, nil
			}
			if err == nil {
				err = fmt.Errorf("no IPv4 answer")
			}
			failures = append(failures, err)
		}
	}
	return nil, fmt.Errorf("interface DNS query failed: %w", errors.Join(failures...))
}
