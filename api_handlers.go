package main

import (
	"encoding/json"
	"net"
	"net/http"
)

func setupAPIHandlers(p *ProxyServer) {
	http.HandleFunc("/api/interfaces", func(w http.ResponseWriter, r *http.Request) {
		ifaces, _ := net.Interfaces()
		var list []map[string]interface{}
		for _, iface := range ifaces {
			addrs, _ := iface.Addrs()
			if len(addrs) > 0 {
				list = append(list, map[string]interface{}{"name": iface.Name, "index": iface.Index})
			}
		}
		json.NewEncoder(w).Encode(list)
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var cfg Config
			json.NewDecoder(r.Body).Decode(&cfg)
			p.mu.Lock()
			p.Config = cfg
			p.mu.Unlock()
			p.saveConfig()
			p.refreshHTTPIfaceInfo()
			w.WriteHeader(http.StatusOK)
			return
		}
		p.mu.RLock()
		json.NewEncoder(w).Encode(p.Config)
		p.mu.RUnlock()
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		p.logMu.Lock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"running":        p.running,
			"port":           p.Config.Port,
			"httpProxyPort":  p.httpProxyPort,
			"httpProxyIface": p.Config.HTTPProxyIface,
			"logs":           p.logBuffer,
		})
		p.logMu.Unlock()
		p.mu.RUnlock()
	})

	http.HandleFunc("/api/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p.clearLogs()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		if err := p.Start(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		p.Stop()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/restart", func(w http.ResponseWriter, r *http.Request) {
		if err := p.Restart(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/autodetect-gfw", func(w http.ResponseWriter, r *http.Request) {
		iface := p.AutoDetectGFWIface()
		p.mu.Lock()
		p.Config.GFWIface = iface
		p.mu.Unlock()
		p.saveConfig()
		json.NewEncoder(w).Encode(map[string]string{"iface": iface})
	})

	http.HandleFunc("/api/detect-system-proxy", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"proxy": detectSystemProxy()})
	})

	http.HandleFunc("/api/autodetect-company", func(w http.ResponseWriter, r *http.Request) {
		iface := p.AutoDetectCompanyIface()
		p.mu.Lock()
		p.Config.CompanyIface = iface
		p.mu.Unlock()
		p.saveConfig()
		json.NewEncoder(w).Encode(map[string]string{"iface": iface})
	})

	http.HandleFunc("/", serveEmbeddedFrontend)
}
