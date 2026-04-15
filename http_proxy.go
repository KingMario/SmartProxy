package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

func (p *ProxyServer) refreshHTTPIfaceInfo() {
	p.mu.RLock()
	iface := p.Config.HTTPProxyIface
	p.mu.RUnlock()

	if iface == "" {
		p.mu.Lock()
		p.httpIfaceIndex = 0
		p.httpIfaceIP = ""
		p.mu.Unlock()
		return
	}

	idx, ip, err := getInterfaceInfo(iface)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.addLog(fmt.Sprintf("Failed to resolve HTTP proxy interface %s: %v", iface, err))
		p.httpIfaceIndex = 0
		p.httpIfaceIP = ""
		return
	}
	p.httpIfaceIndex = idx
	p.httpIfaceIP = ip
}

func (p *ProxyServer) startHTTPProxy() error {
	p.refreshHTTPIfaceInfo()
	addr := fmt.Sprintf("0.0.0.0:%d", httpProxyPortConst)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.httpListener = ln
	p.httpProxyPort = httpProxyPortConst
	p.mu.Unlock()
	p.addLog(fmt.Sprintf("HTTP proxy started on 0.0.0.0:%d", p.httpProxyPort))

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("HTTP proxy accept error: %v", err)
				return
			}
			go p.handleHTTPConnection(conn)
		}
	}()
	return nil
}

func (p *ProxyServer) handleHTTPConnect(client net.Conn, target string) {
	host, _, _ := net.SplitHostPort(target)
	remote, err := p.dialRemote(host, target)
	if err != nil {
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(remote, client)
		if tcp, ok := remote.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(client, remote)
		if tcp, ok := client.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()
	wg.Wait()
}

func (p *ProxyServer) handleHTTPConnection(client net.Conn) {
	defer client.Close()
	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		host := req.Host
		if host != "" && !strings.Contains(host, ":") {
			host = net.JoinHostPort(host, "443")
		}
		if host == "" {
			return
		}
		p.handleHTTPConnect(client, host)
		return
	}

	target := req.Host
	if target == "" {
		return
	}
	if !strings.Contains(target, ":") {
		target = net.JoinHostPort(target, "80")
	}
	host, _, _ := net.SplitHostPort(target)
	remote, err := p.dialRemote(host, target)
	if err != nil {
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	req.RequestURI = ""
	req.URL.Scheme = ""
	req.URL.Host = ""
	if err := req.Write(remote); err != nil {
		return
	}
	if req.Body != nil {
		req.Body.Close()
	}

	io.Copy(client, remote)
}
