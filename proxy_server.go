package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type ProxyServer struct {
	Config         Config
	GFWDomains     map[string]bool
	IfaceIndices   map[string]int
	IfaceIPs       map[string]string
	listener       net.Listener
	running        bool
	mu             sync.RWMutex
	logBuffer      []string
	logMu          sync.Mutex
	configPath     string
	onStatusChange func(running bool)
	httpListener   net.Listener
	httpProxyPort  int
	httpIfaceIndex int
	httpIfaceIP    string
}

const httpProxyPortConst = 17890

func (p *ProxyServer) addLog(msg string) {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	ts := time.Now().Format("15:04:05")
	p.logBuffer = append(p.logBuffer, fmt.Sprintf("[%s] %s", ts, msg))
	if len(p.logBuffer) > 100 {
		p.logBuffer = p.logBuffer[1:]
	}
	log.Println(msg)
}

func (p *ProxyServer) clearLogs() {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	p.logBuffer = nil
}

func (p *ProxyServer) Start() error {
	systemProxy := detectSystemProxy()
	ignoredConfiguredGFWProxy := false

	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	if systemProxy == "" && p.Config.GFWProxy != "" {
		p.Config.GFWProxy = ""
		ignoredConfiguredGFWProxy = true
	}

	p.IfaceIndices = make(map[string]int)
	p.IfaceIPs = make(map[string]string)
	for _, name := range []string{p.Config.DefaultIface, p.Config.GFWIface, p.Config.CompanyIface, p.Config.HTTPProxyIface} {
		if name == "" {
			continue
		}
		idx, ip, err := getInterfaceInfo(name)
		if err == nil {
			p.IfaceIndices[name] = idx
			p.IfaceIPs[name] = ip
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p.Config.Port))
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.listener = ln
	p.running = true
	p.mu.Unlock()

	if ignoredConfiguredGFWProxy {
		p.addLog("Ignored configured GFW upstream proxy because no global proxy was detected")
	}

	if p.onStatusChange != nil {
		p.onStatusChange(true)
	}

	p.loadGFWList()
	p.addLog(fmt.Sprintf("SOCKS5 Proxy started on 0.0.0.0:%d", p.Config.Port))

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go p.handleConnection(conn)
		}
	}()
	return nil
}

func (p *ProxyServer) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *ProxyServer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener != nil {
		p.listener.Close()
		p.listener = nil
	}
	p.running = false
	if p.onStatusChange != nil {
		p.onStatusChange(false)
	}
	p.addLog("Proxy server stopped")
}

func (p *ProxyServer) Restart() error {
	wasRunning := p.IsRunning()
	if wasRunning {
		p.addLog("Restarting proxy server")
		p.Stop()
		time.Sleep(100 * time.Millisecond)
	}

	if err := p.Start(); err != nil {
		return err
	}

	if wasRunning {
		p.addLog("Proxy server restarted")
	}

	return nil
}

func (p *ProxyServer) handleConnection(client net.Conn) {
	defer client.Close()
	buf := make([]byte, 256)
	if _, err := io.ReadFull(client, buf[:2]); err != nil || buf[0] != 0x05 {
		return
	}
	nmethods := int(buf[1])
	if _, err := io.ReadFull(client, buf[:nmethods]); err != nil {
		return
	}
	client.Write([]byte{0x05, 0x00})
	if _, err := io.ReadFull(client, buf[:4]); err != nil || buf[0] != 0x05 {
		return
	}
	var host string
	switch buf[3] {
	case 0x01:
		if _, err := io.ReadFull(client, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03:
		if _, err := io.ReadFull(client, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(client, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	default:
		return
	}
	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	targetAddr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	remote, err := p.dialRemote(host, targetAddr)
	if err != nil {
		client.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
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
