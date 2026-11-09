package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

// Config represents the proxy configuration
type Config struct {
	Port                int      `json:"port"`
	DefaultIface        string   `json:"defaultIface"`
	GFWIface            string   `json:"gfwIface"`
	CompanyIface        string   `json:"companyIface"`
	GFWListURL          string   `json:"gfwlistUrl"`
	CompanyDomains      []string `json:"companyDomains"`
	BypassDomains       []string `json:"bypassDomains"`
	ExtraGFWDomains     []string `json:"extraGfwDomains"`
	AutoStart           bool     `json:"autoStart"`
	HTTPProxyIface      string   `json:"httpProxyIface"`
	HTTPProxyDNS        []string `json:"httpProxyDNS"`
	GFWProxy            string   `json:"gfwProxy"`
	VerboseLog          bool     `json:"verboseLog"`
	AutoUpdateGFWList   bool     `json:"autoUpdateGfwList"`
	DefaultDNS          string   `json:"defaultDNS"`
	DefaultDNSBootstrap []string `json:"defaultDNSBootstrap"`
	GFWDNS              string   `json:"gfwDNS"`
	GFWDNSBootstrap     []string `json:"gfwDNSBootstrap"`
	CompanyDNS          string   `json:"companyDNS"`
	CompanyDNSBootstrap []string `json:"companyDNSBootstrap"`
}

func (p *ProxyServer) saveConfig() error {
	p.mu.RLock()
	data, err := json.MarshalIndent(p.Config, "", "  ")
	p.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(p.configPath, data, 0644)
}

func (p *ProxyServer) loadConfig() error {
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return err
	}
	cfg := p.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if err := validateDNSConfig(cfg); err != nil {
		return fmt.Errorf("invalid DNS configuration: %w", err)
	}
	if err := validateHTTPProxyConfig(cfg); err != nil {
		return err
	}
	p.Config = cfg
	return nil
}

func validateHTTPProxyConfig(cfg Config) error {
	if cfg.Port == httpProxyPortConst || cfg.Port == directHTTPProxyPortConst {
		return fmt.Errorf("SOCKS5 port must differ from HTTP ports 17890 and 17891")
	}
	for _, server := range cfg.HTTPProxyDNS {
		ip := net.ParseIP(strings.TrimSpace(server))
		if ip == nil || ip.To4() == nil || ip.IsUnspecified() {
			return fmt.Errorf("HTTP proxy DNS servers must be IPv4 addresses")
		}
	}
	return nil
}
