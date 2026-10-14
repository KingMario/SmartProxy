package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (p *ProxyServer) loadGFWList() error {
	p.mu.RLock()
	url := p.Config.GFWListURL
	p.mu.RUnlock()

	var raw []byte
	var err error

	if strings.HasPrefix(url, "@") || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		path := strings.TrimPrefix(url, "@")
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(p.configPath), path)
		}
		raw, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		raw, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	content := ""
	if err == nil {
		content = string(decoded)
	} else {
		content = string(raw)
	}

	p.mu.Lock()
	p.GFWDomains = make(map[string]bool)
	domainRegex := regexp.MustCompile(`([A-Za-z0-9.-]+\.[A-Za-z]{2,})$`)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "@@") {
			continue
		}
		line = strings.TrimLeft(line, "|.")
		for _, sep := range []string{"/", "^", "*", "?"} {
			if idx := strings.Index(line, sep); idx != -1 {
				line = line[:idx]
			}
		}
		line = strings.Trim(line, ".")
		match := domainRegex.FindStringSubmatch(line)
		if len(match) > 1 {
			p.GFWDomains[strings.ToLower(match[1])] = true
		}
	}
	p.mu.Unlock()

	p.addLog(fmt.Sprintf("Loaded %d domains from GFWList", len(p.GFWDomains)))
	return nil
}

func (p *ProxyServer) isGFWDomain(host string) bool {
	host = strings.ToLower(host)
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isGFWDomainLocked(host)
}

func (p *ProxyServer) isGFWDomainLocked(host string) bool {
	if p.GFWDomains[host] {
		return true
	}
	for _, domain := range p.Config.ExtraGFWDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	parts := strings.Split(host, ".")
	for i := 0; i < len(parts)-1; i++ {
		suffix := strings.Join(parts[i:], ".")
		if p.GFWDomains[suffix] {
			return true
		}
	}
	return false
}

func (p *ProxyServer) isBypassDomain(host string) bool {
	host = strings.ToLower(host)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, domain := range p.Config.BypassDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func (p *ProxyServer) isCompanyDomain(host string) bool {
	host = strings.ToLower(host)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, domain := range p.Config.CompanyDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func (p *ProxyServer) selectIface(host string) string {
	return p.selectRoute(host).iface
}

type outboundRoute struct {
	iface     string
	dns       string
	bootstrap []string
	upstream  string
}

func matchesDomains(host string, domains []string) bool {
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
		if domain != "" && (host == domain || strings.HasSuffix(host, "."+domain)) {
			return true
		}
	}
	return false
}

// Take one configuration snapshot so DNS and the connection choose the same route.
func (p *ProxyServer) selectRoute(host string) outboundRoute {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg := p.Config
	route := outboundRoute{iface: cfg.DefaultIface, dns: cfg.DefaultDNS, bootstrap: cfg.DefaultDNSBootstrap}
	switch {
	case matchesDomains(host, cfg.BypassDomains):
	case matchesDomains(host, cfg.CompanyDomains) && cfg.CompanyIface != "":
		route = outboundRoute{iface: cfg.CompanyIface, dns: cfg.CompanyDNS, bootstrap: cfg.CompanyDNSBootstrap}
	case net.ParseIP(host) != nil:
	case p.isGFWDomainLocked(host):
		route = outboundRoute{iface: cfg.GFWIface, dns: cfg.GFWDNS, bootstrap: cfg.GFWDNSBootstrap, upstream: cfg.GFWProxy}
	}
	route.bootstrap = append([]string(nil), route.bootstrap...)
	return route
}
