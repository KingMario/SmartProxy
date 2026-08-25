package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const maxGFWListSize = 5 * 1024 * 1024

var officialGFWListURL = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"

func newGFWListHTTPClient(interfaceName string) (*http.Client, bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	if interfaceName == "" {
		return client, false, nil
	}

	interfaceIndex, localIP, err := getInterfaceInfo(interfaceName)
	if err != nil {
		return client, false, fmt.Errorf("resolve GFW interface %q: %w", interfaceName, err)
	}

	newDialer := func(useLocalIP bool) *net.Dialer {
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		if useLocalIP && localIP != "" {
			dialer.LocalAddr = &net.TCPAddr{IP: net.ParseIP(localIP)}
		}
		dialer.Control = func(network, address string, connection syscall.RawConn) error {
			return connection.Control(func(fileDescriptor uintptr) {
				bindSocketToInterface(fileDescriptor, network, interfaceIndex)
			})
		}
		return dialer
	}

	dialer := newDialer(true)
	dnsDialer := newDialer(false)
	dialer.Resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dnsDialer.DialContext(ctx, network, address)
		},
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, address)
	}
	client.Transport = transport
	return client, true, nil
}

func (p *ProxyServer) updateGFWList() (bool, error) {
	p.mu.RLock()
	configPath := p.configPath
	configuredInterface := p.Config.GFWIface
	configuredListURL := p.Config.GFWListURL
	p.mu.RUnlock()

	client, bound, interfaceErr := newGFWListHTTPClient(configuredInterface)
	if interfaceErr != nil {
		p.addLog(fmt.Sprintf("GFWList update: %v; using the default network", interfaceErr))
	} else if bound {
		p.addLog(fmt.Sprintf("GFWList update: using GFW interface %s", configuredInterface))
	}

	localPath := filepath.Join(filepath.Dir(configPath), "gfwlist.txt")
	updated, err := updateGFWList(officialGFWListURL, localPath, client)
	if err != nil {
		return false, err
	}
	if !updated {
		p.addLog("GFWList is already up to date")
		return false, nil
	}

	if isLocalGFWList(configuredListURL, configPath, localPath) {
		if err := p.loadGFWList(); err != nil {
			return true, fmt.Errorf("reload GFWList: %w", err)
		}
	}
	p.addLog("GFWList updated successfully")
	return true, nil
}

func (p *ProxyServer) updateGFWListOnStartup() (bool, error) {
	p.mu.RLock()
	autoUpdate := p.Config.AutoUpdateGFWList
	p.mu.RUnlock()
	if !autoUpdate {
		return false, nil
	}

	return p.updateGFWList()
}

func isLocalGFWList(listURL, configPath, localPath string) bool {
	if strings.HasPrefix(listURL, "http://") || strings.HasPrefix(listURL, "https://") {
		return false
	}
	listPath := strings.TrimPrefix(listURL, "@")
	if !filepath.IsAbs(listPath) {
		listPath = filepath.Join(filepath.Dir(configPath), listPath)
	}
	return filepath.Clean(listPath) == filepath.Clean(localPath)
}

func updateGFWList(sourceURL, localPath string, client *http.Client) (bool, error) {
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Get(sourceURL)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("download GFWList: unexpected HTTP status %s", response.Status)
	}

	latest, err := io.ReadAll(io.LimitReader(response.Body, maxGFWListSize+1))
	if err != nil {
		return false, fmt.Errorf("download GFWList: %w", err)
	}
	if len(latest) > maxGFWListSize {
		return false, fmt.Errorf("download GFWList: file exceeds %d bytes", maxGFWListSize)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(latest))
	if err != nil || !strings.Contains(string(decoded), "[AutoProxy") {
		return false, fmt.Errorf("download GFWList: invalid list content")
	}

	current, err := os.ReadFile(localPath)
	if err == nil && bytes.Equal(current, latest) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	directory := filepath.Dir(localPath)
	temporaryFile, err := os.CreateTemp(directory, filepath.Base(localPath)+".*")
	if err != nil {
		return false, err
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	if _, err := temporaryFile.Write(latest); err != nil {
		temporaryFile.Close()
		return false, err
	}
	if err := temporaryFile.Chmod(0644); err != nil {
		temporaryFile.Close()
		return false, err
	}
	if err := temporaryFile.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(temporaryPath, localPath); err != nil {
		return false, err
	}

	return true, nil
}
