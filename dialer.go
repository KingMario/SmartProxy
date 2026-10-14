package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

func (p *ProxyServer) dialViaProxy(proxyAddr, targetAddr string) (net.Conn, error) {
	proxyAddr = strings.TrimPrefix(proxyAddr, "http://")
	conn, err := net.DialTimeout("tcp", proxyAddr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetAddr, targetAddr)

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	return conn, nil
}

func (p *ProxyServer) dialRemote(host, targetAddr string) (net.Conn, error) {
	route := p.selectRoute(host)
	p.mu.RLock()
	verbose := p.Config.VerboseLog
	p.mu.RUnlock()

	if route.upstream != "" {
		if verbose {
			p.addLog(fmt.Sprintf("[GFW] %s -> Upstream Proxy %s", targetAddr, route.upstream))
		}
		return p.dialViaProxy(route.upstream, targetAddr)
	}

	targetIface := route.iface
	info, err := p.resolveInterface(targetIface, false)
	if err != nil {
		err = fmt.Errorf("resolve interface %q for %s: %w", targetIface, targetAddr, err)
		p.addLog(err.Error())
		return nil, err
	}

	if verbose {
		p.addLog(fmt.Sprintf("[DIR] %s -> Interface %s", targetAddr, targetIface))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := p.dialResolved(ctx, targetAddr, route, info, verbose)
	if err != nil && targetIface != "" && isInterfaceError(err) {
		current, refreshErr := p.resolveInterface(targetIface, true)
		if refreshErr != nil {
			return nil, fmt.Errorf("refresh interface %q: %w", targetIface, refreshErr)
		}
		// Retry only when a fresh snapshot identifies a different binding.
		if current != info {
			return p.dialResolved(ctx, targetAddr, route, current, verbose)
		}
	}
	return conn, err
}

func dialWithInterface(targetAddr string, info interfaceInfo) (net.Conn, error) {
	return dialWithInterfaceContext(context.Background(), targetAddr, info)
}

func dialWithInterfaceContext(ctx context.Context, targetAddr string, info interfaceInfo) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	if info.ip != "" {
		dialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(info.ip)}
	}
	if info.index != 0 {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			var bindErr error
			if err := c.Control(func(fd uintptr) {
				bindErr = bindSocketToInterface(fd, network, info.index)
			}); err != nil {
				return err
			}
			return bindErr
		}
	}
	return dialer.DialContext(ctx, "tcp", targetAddr)
}

func (p *ProxyServer) dialResolved(ctx context.Context, targetAddr string, route outboundRoute, info interfaceInfo, verbose bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) != nil {
		return dialWithInterfaceContext(ctx, targetAddr, info)
	}
	ips, err := p.dnsResolver().lookup(ctx, host, route, info)
	if err != nil {
		p.addLog(fmt.Sprintf("[DNS] %s via interface %s: %v", host, route.iface, err))
		return nil, err
	}
	if verbose {
		p.addLog(fmt.Sprintf("[DNS] %s -> %s (Interface %s)", host, strings.Join(ips, ", "), route.iface))
	}
	var failures []error
	for i, ip := range ips {
		attempt := ctx
		cancel := func() {}
		if i < len(ips)-1 {
			timeout := 5 * time.Second
			if deadline, ok := ctx.Deadline(); ok {
				if share := time.Until(deadline) / time.Duration(len(ips)-i); share < timeout {
					timeout = share
				}
			}
			attempt, cancel = context.WithTimeout(ctx, timeout)
		}
		conn, err := dialWithInterfaceContext(attempt, net.JoinHostPort(ip, port), info)
		cancel()
		if err == nil {
			return conn, nil
		}
		failures = append(failures, err)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("connect %s: %w", targetAddr, errors.Join(failures...))
}
