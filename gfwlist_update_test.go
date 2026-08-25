package main

import (
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewGFWListHTTPClientBindsConfiguredInterface(t *testing.T) {
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}

	configuredInterface := ""
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 {
			configuredInterface = iface.Name
			break
		}
	}
	if configuredInterface == "" {
		t.Skip("no active network interface available")
	}

	client, bound, err := newGFWListHTTPClient(configuredInterface)
	if err != nil {
		t.Fatalf("newGFWListHTTPClient() error = %v", err)
	}
	if !bound {
		t.Fatalf("newGFWListHTTPClient(%q) bound = false, want true", configuredInterface)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.DialContext == nil {
		t.Fatal("configured GFWList client does not provide an interface-bound dialer")
	}
}

func TestUpdateGFWListReplacesLocalListWhenOfficialListChanges(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "gfwlist.txt")
	if err := os.WriteFile(localPath, []byte(base64.StdEncoding.EncodeToString([]byte("[AutoProxy 0.2.9]\n||old.example^\n"))), 0600); err != nil {
		t.Fatal(err)
	}

	latest := base64.StdEncoding.EncodeToString([]byte("[AutoProxy 0.2.9]\n||new.example^\n"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gfwlist.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(latest))
	}))
	defer server.Close()

	updated, err := updateGFWList(server.URL+"/gfwlist.txt", localPath, server.Client())
	if err != nil {
		t.Fatalf("updateGFWList() error = %v", err)
	}
	if !updated {
		t.Fatal("updateGFWList() updated = false, want true")
	}

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != latest {
		t.Fatalf("local list = %q, want %q", got, latest)
	}
}

func TestUpdateGFWListKeepsLocalListWhenOfficialListIsInvalid(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "gfwlist.txt")
	original := base64.StdEncoding.EncodeToString([]byte("[AutoProxy 0.2.9]\n||working.example^\n"))
	if err := os.WriteFile(localPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not a valid GFWList"))
	}))
	defer server.Close()

	updated, err := updateGFWList(server.URL, localPath, server.Client())
	if err == nil {
		t.Fatal("updateGFWList() error = nil, want validation error")
	}
	if updated {
		t.Fatal("updateGFWList() updated = true, want false")
	}

	got, readErr := os.ReadFile(localPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Fatalf("local list changed to %q, want %q", got, original)
	}
}

func TestProxyServerUpdateGFWListDownloadsOfficialListToItsConfigDirectory(t *testing.T) {
	configDirectory := t.TempDir()
	latest := base64.StdEncoding.EncodeToString([]byte("[AutoProxy 0.2.9]\n||new.example^\n"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(latest))
	}))
	defer server.Close()

	previousSource := officialGFWListURL
	officialGFWListURL = server.URL
	defer func() { officialGFWListURL = previousSource }()

	proxy := &ProxyServer{
		configPath: filepath.Join(configDirectory, "config.json"),
		Config: Config{
			AutoUpdateGFWList: true,
			GFWListURL:        "gfwlist.txt",
		},
	}

	updated, err := proxy.updateGFWListOnStartup()
	if err != nil {
		t.Fatalf("updateGFWList() error = %v", err)
	}
	if !updated {
		t.Fatal("updateGFWList() updated = false, want true")
	}

	got, err := os.ReadFile(filepath.Join(configDirectory, "gfwlist.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != latest {
		t.Fatalf("local list = %q, want %q", got, latest)
	}
}

func TestUpdateGFWListOnStartupSkipsNetworkRequestWhenDisabled(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte("unexpected request"))
	}))
	defer server.Close()

	previousSource := officialGFWListURL
	officialGFWListURL = server.URL
	defer func() { officialGFWListURL = previousSource }()

	proxy := &ProxyServer{
		configPath: filepath.Join(t.TempDir(), "config.json"),
		Config: Config{
			AutoUpdateGFWList: false,
			GFWListURL:        "gfwlist.txt",
		},
	}

	updated, err := proxy.updateGFWListOnStartup()
	if err != nil {
		t.Fatalf("updateGFWListOnStartup() error = %v", err)
	}
	if updated {
		t.Fatal("updateGFWListOnStartup() updated = true, want false")
	}
	if requests != 0 {
		t.Fatalf("GFWList requests = %d, want 0", requests)
	}
}

func TestGFWListUpdateHandlerRejectsNonPostRequests(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/gfwlist/update", nil)
	recorder := httptest.NewRecorder()

	gfwListUpdateHandler(&ProxyServer{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
