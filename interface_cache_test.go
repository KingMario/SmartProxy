package main

import (
	"errors"
	"sync"
	"syscall"
	"testing"
)

func TestInterfaceCacheLifecycle(t *testing.T) {
	calls := 0
	index, ip := 6, "172.16.0.1"
	var lookupErr error
	c := newInterfaceCache(func(name string) (int, string, error) {
		calls++
		return index, ip, lookupErr
	})
	first, err := c.resolve("utun6", false)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		if _, err := c.resolve("utun6", false); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("stable interface queried %d times", calls)
	}
	index, ip = 9, "172.16.0.2"
	c.invalidate()
	next, err := c.resolve("utun6", false)
	if err != nil || next == first || next.index != 9 || next.ip != ip {
		t.Fatalf("stale state: %+v, %v", next, err)
	}
	lookupErr = errors.New("VPN absent")
	c.invalidate()
	if _, err := c.resolve("utun6", false); err == nil {
		t.Fatal("missing VPN accepted")
	}
	lookupErr = nil
	if _, err := c.resolve("utun6", false); err != nil {
		t.Fatalf("late VPN: %v", err)
	}
	index = 10
	forced, err := c.resolve("utun6", true)
	if err != nil || forced.index != 10 {
		t.Fatalf("forced refresh: %+v %v", forced, err)
	}
	lookupErr = errors.New("VPN removed")
	if _, err := c.resolve("utun6", true); err == nil {
		t.Fatal("forced refresh accepted removed VPN")
	}
	if _, err := c.resolve("utun6", false); err == nil {
		t.Fatal("retained stale cache after lookup failed")
	}
}

func TestInterfaceCacheConcurrentMiss(t *testing.T) {
	calls := 0
	c := newInterfaceCache(func(string) (int, string, error) { calls++; return 6, "172.16.0.1", nil })
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, _ = c.resolve("utun6", false) })
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("concurrent misses queried %d times", calls)
	}
}

func TestInterfaceErrorClassification(t *testing.T) {
	for _, err := range []error{syscall.ENETDOWN, syscall.ENETUNREACH, syscall.EADDRNOTAVAIL, syscall.ENXIO} {
		if !isInterfaceError(err) {
			t.Fatalf("not recognized: %v", err)
		}
	}
	for _, err := range []error{syscall.ECONNREFUSED, syscall.ETIMEDOUT, errors.New("DNS failure")} {
		if isInterfaceError(err) {
			t.Fatalf("unrelated failure: %v", err)
		}
	}
}

func TestInterfaceCacheInvalidationDuringLookup(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	calls := 0
	c := newInterfaceCache(func(string) (int, string, error) {
		calls++
		if calls == 1 {
			close(entered)
			<-release
		}
		return calls, "", nil
	})
	resolved := make(chan struct{})
	go func() { _, _ = c.resolve("utun6", false); close(resolved) }()
	<-entered
	invalidated := make(chan struct{})
	go func() { c.invalidate(); close(invalidated) }()
	close(release)
	<-resolved
	<-invalidated
	info, err := c.resolve("utun6", false)
	if err != nil || info.index != 2 {
		t.Fatalf("in-flight lookup survived invalidation: %+v %v", info, err)
	}
}

func TestInterfaceCacheRefreshKnownInterfaces(t *testing.T) {
	calls := 0
	c := newInterfaceCache(func(string) (int, string, error) { calls++; return calls, "", nil })
	_, _ = c.resolve("company", false)
	_, _ = c.resolve("gfw", false)
	c.invalidate()
	c.invalidate()
	c.refresh()
	c.refresh()
	if calls != 4 {
		t.Fatalf("refresh queries = %d, want 4", calls)
	}
}
