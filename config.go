package main

import (
	"encoding/json"
	"os"
)

// Config represents the proxy configuration
type Config struct {
	Port            int      `json:"port"`
	DefaultIface    string   `json:"defaultIface"`
	GFWIface        string   `json:"gfwIface"`
	CompanyIface    string   `json:"companyIface"`
	GFWListURL      string   `json:"gfwlistUrl"`
	CompanyDomains  []string `json:"companyDomains"`
	BypassDomains   []string `json:"bypassDomains"`
	ExtraGFWDomains []string `json:"extraGfwDomains"`
	AutoStart       bool     `json:"autoStart"`
	HTTPProxyIface  string   `json:"httpProxyIface"`
	GFWProxy        string   `json:"gfwProxy"`
	VerboseLog      bool     `json:"verboseLog"`
}

func (p *ProxyServer) saveConfig() error {
	data, err := json.MarshalIndent(p.Config, "", "  ")
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
	return json.Unmarshal(data, &p.Config)
}
