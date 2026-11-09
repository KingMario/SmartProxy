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

func (p *ProxyServer) startHTTPProxy() error {
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

type httpProxyDial func(host, target string) (net.Conn, error)

func (p *ProxyServer) startDirectHTTPProxy() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", directHTTPProxyPortConst))
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.directHTTPListener, p.directHTTPProxyPort = ln, directHTTPProxyPortConst
	p.mu.Unlock()
	p.addLog("Fixed-interface HTTP proxy started on 0.0.0.0:17891")
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go p.handleHTTPConnectionWithDial(conn, p.dialDirectHTTP)
		}
	}()
	return nil
}

func (p *ProxyServer) handleHTTPConnect(client net.Conn, reader io.Reader, target string, dial httpProxyDial) {
	host, _, _ := net.SplitHostPort(target)
	remote, err := dial(host, target)
	if err != nil {
		p.addLog(fmt.Sprintf("HTTP CONNECT %s: %v", target, err))
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(remote, reader)
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
	p.handleHTTPConnectionWithDial(client, p.dialRemote)
}

func (p *ProxyServer) handleHTTPConnectionWithDial(client net.Conn, dial httpProxyDial) {
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
		p.handleHTTPConnect(client, br, host, dial)
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
	remote, err := dial(host, target)
	if err != nil {
		p.addLog(fmt.Sprintf("HTTP %s: %v", target, err))
		client.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	req.RequestURI = ""
	req.URL.Scheme = ""
	req.URL.Host = ""
	// This handler serves one HTTP request per client connection. Ask the
	// origin to close so clients do not reuse a connection we cannot read again.
	req.Close = true
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	if err := req.Write(remote); err != nil {
		return
	}
	if req.Body != nil {
		req.Body.Close()
	}

	io.Copy(client, remote)
}
