package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	p.Config = cfg
	return nil
}
