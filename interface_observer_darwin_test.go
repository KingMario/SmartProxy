//go:build darwin && cgo

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNativeInterfaceObserverNotification(t *testing.T) {
	calls := 0
	cache := newInterfaceCache(func(string) (int, string, error) { calls++; return calls, "", nil })
	changed := make(chan struct{}, 1)
	stop, err := observeInterfaceChanges(func() {
		cache.invalidate()
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	first, err := cache.resolve("test", false)
	if err != nil {
		t.Fatal(err)
	}
	// Post a notification only; no network configuration is changed.
	key := fmt.Sprintf("State:/Network/Interface/SmartProxyTest%d/IPv4", os.Getpid())
	cmd := exec.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader("notify " + key + "\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("notify: %v: %s", err, output)
	}
	if strings.Contains(string(output), "Permission denied") {
		t.Skip("system notification posting requires privileges; observer registration succeeded")
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("notify: %s", output)
	}
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("native notification was not delivered")
	}
	next, err := cache.resolve("test", false)
	if err != nil || next == first {
		t.Fatalf("notification retained stale cache: %+v, %v", next, err)
	}
	stop()
	stop()
}

func TestDialRemoteRefreshesStaleBinding(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	calls := 0
	cache := newInterfaceCache(func(string) (int, string, error) {
		calls++
		return getInterfaceInfo("lo0")
	})
	cache.entries["lo0"] = interfaceInfo{index: 999999, ip: "192.0.2.1"}
	p := &ProxyServer{Config: Config{DefaultIface: "lo0"}, interfaceCache: cache}
	conn, err := p.dialRemote("127.0.0.1", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if calls != 1 {
		t.Fatalf("refresh queries = %d, want 1", calls)
	}
}

func TestInterfaceMonitorLifecycle(t *testing.T) {
	for range 5 {
		p := &ProxyServer{Config: Config{DefaultIface: "lo0"}}
		stop := p.startInterfaceMonitor()
		if p.interfaceCache == nil {
			t.Fatal("native observer was not registered")
		}
		stop()
		stop()
	}
}
