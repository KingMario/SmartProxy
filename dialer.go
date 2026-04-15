package main

import (
	"bufio"
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
	isGFW := p.isGFWDomain(host)
	p.mu.RLock()
	gfwProxy := p.Config.GFWProxy
	verbose := p.Config.VerboseLog
	p.mu.RUnlock()

	if isGFW && gfwProxy != "" {
		if verbose {
			p.addLog(fmt.Sprintf("[GFW] %s -> Upstream Proxy %s", targetAddr, gfwProxy))
		}
		return p.dialViaProxy(gfwProxy, targetAddr)
	}

	targetIface := p.selectIface(host)
	p.mu.RLock()
	ifIndex := p.IfaceIndices[targetIface]
	localIP := p.IfaceIPs[targetIface]
	p.mu.RUnlock()

	if verbose {
		p.addLog(fmt.Sprintf("[DIR] %s -> Interface %s", targetAddr, targetIface))
	}

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	if localIP != "" {
		dialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(localIP)}
	}
	if ifIndex != 0 {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				bindSocketToInterface(fd, network, ifIndex)
			})
		}
	}

	return dialer.Dial("tcp", targetAddr)
}
