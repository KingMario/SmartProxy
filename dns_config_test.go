package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigAPIPreservesDNSForOldClientsAndRejectsInvalidDNS(t *testing.T) {
	previousMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	defer func() { http.DefaultServeMux = previousMux }()
	p := &ProxyServer{configPath: filepath.Join(t.TempDir(), "config.json"), Config: Config{
		Port: 1080, DefaultDNS: "https://resolver.test/dns-query", DefaultDNSBootstrap: []string{"192.0.2.1"},
		HTTPProxyDNS: []string{"114.114.114.114"},
	}}
	setupAPIHandlers(p)
	post := func(body string) int {
		recorder := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body)))
		return recorder.Code
	}
	if code := post(`{"port":1081}`); code != http.StatusOK {
		t.Fatalf("legacy save: HTTP %d", code)
	}
	if p.Config.Port != 1081 || p.Config.DefaultDNS != "https://resolver.test/dns-query" || len(p.Config.DefaultDNSBootstrap) != 1 {
		t.Fatalf("legacy save lost DNS: %+v", p.Config)
	}
	if len(p.Config.HTTPProxyDNS) != 1 || p.Config.HTTPProxyDNS[0] != "114.114.114.114" {
		t.Fatal("legacy save lost fixed-interface DNS")
	}
	for _, body := range []string{`{invalid`, `{"port":1082,"defaultDNS":"http://resolver.test"}`, `{"port":17891}`, `{"port":1082,"httpProxyDNS":["resolver.test"]}`} {
		if code := post(body); code != http.StatusBadRequest {
			t.Fatalf("invalid save: HTTP %d", code)
		}
		if p.Config.Port != 1081 {
			t.Fatal("invalid request changed live configuration")
		}
	}
	if code := post(`{"port":1081,"defaultDNS":"system","defaultDNSBootstrap":[],"httpProxyDNS":[]}`); code != http.StatusOK {
		t.Fatalf("explicit reset: HTTP %d", code)
	}
	if p.Config.DefaultDNS != "system" || len(p.Config.DefaultDNSBootstrap) != 0 {
		t.Fatal("explicit DNS reset ignored")
	}
	if len(p.Config.HTTPProxyDNS) != 0 {
		t.Fatal("explicit interface DNS reset ignored")
	}
}
