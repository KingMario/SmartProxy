package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func detectSystemProxy() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("scutil", "--proxy").Output()
		if err != nil {
			return ""
		}
		s := string(out)
		var host, port string
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "HTTPProxy :") {
				host = strings.TrimSpace(strings.TrimPrefix(line, "HTTPProxy :"))
			} else if strings.HasPrefix(line, "HTTPPort :") {
				port = strings.TrimSpace(strings.TrimPrefix(line, "HTTPPort :"))
			}
		}
		if host != "" && port != "" {
			return fmt.Sprintf("%s:%s", host, port)
		}
	} else if runtime.GOOS == "windows" {
		out, err := exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings", "/v", "ProxyServer").Output()
		if err == nil {
			s := string(out)
			parts := strings.Fields(s)
			for i, p := range parts {
				if p == "REG_SZ" && i+1 < len(parts) {
					server := parts[i+1]
					if strings.Contains(server, ";") {
						for _, sub := range strings.Split(server, ";") {
							if strings.HasPrefix(sub, "http=") {
								return strings.TrimPrefix(sub, "http=")
							}
						}
					}
					return server
				}
			}
		}
	}
	return ""
}

func getInterfaceInfo(ifaceName string) (int, string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return 0, "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return iface.Index, "", nil
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return iface.Index, ipnet.IP.String(), nil
			}
		}
	}
	return iface.Index, "", nil
}

func (p *ProxyServer) AutoDetectGFWIface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		p.addLog(fmt.Sprintf("Failed to list interfaces: %v", err))
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		idx, ip, err := getInterfaceInfo(iface.Name)
		if err != nil || ip == "" {
			continue
		}

		p.addLog(fmt.Sprintf("Testing interface %s (%s)...", iface.Name, ip))

		dialer := &net.Dialer{
			Timeout: 3 * time.Second,
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					bindSocketToInterface(fd, network, idx)
				})
			},
		}

		conn, err := dialer.Dial("tcp", "www.google.com:80")
		if err == nil {
			conn.Close()
			p.addLog(fmt.Sprintf("Interface %s is working for GFW", iface.Name))
			return iface.Name
		}
		p.addLog(fmt.Sprintf("Interface %s failed: %v", iface.Name, err))
	}

	p.addLog("No interface can reach www.google.com, setting GFW Interface to None")
	return ""
}

func (p *ProxyServer) AutoDetectCompanyIface() string {
	p.mu.RLock()
	if len(p.Config.CompanyDomains) == 0 {
		p.mu.RUnlock()
		p.addLog("No Company Domains configured, cannot detect Company Interface")
		return ""
	}
	targetDomain := p.Config.CompanyDomains[0]
	p.mu.RUnlock()

	ifaces, err := net.Interfaces()
	if err != nil {
		p.addLog(fmt.Sprintf("Failed to list interfaces: %v", err))
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		idx, ip, err := getInterfaceInfo(iface.Name)
		if err != nil || ip == "" {
			continue
		}

		p.addLog(fmt.Sprintf("Testing interface %s (%s) for company domain %s...", iface.Name, ip, targetDomain))

		dialer := &net.Dialer{
			Timeout: 3 * time.Second,
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					bindSocketToInterface(fd, network, idx)
				})
			},
		}

		conn, err := dialer.Dial("tcp", net.JoinHostPort(targetDomain, "80"))
		if err == nil {
			conn.Close()
			p.addLog(fmt.Sprintf("Interface %s is working for Company VPN", iface.Name))
			return iface.Name
		}
		conn, err = dialer.Dial("tcp", net.JoinHostPort(targetDomain, "443"))
		if err == nil {
			conn.Close()
			p.addLog(fmt.Sprintf("Interface %s is working for Company VPN", iface.Name))
			return iface.Name
		}
		p.addLog(fmt.Sprintf("Interface %s failed for %s: %v", iface.Name, targetDomain, err))
	}

	p.addLog(fmt.Sprintf("No interface can reach %s, setting Company Interface to None", targetDomain))
	return ""
}
