package main

import (
	"errors"
	"sync"
	"syscall"
	"time"
)

type interfaceInfo struct {
	index int
	ip    string
}

type interfaceCache struct {
	mu      sync.Mutex
	entries map[string]interfaceInfo
	names   map[string]struct{}
	lookup  func(string) (int, string, error)
}

func newInterfaceCache(lookup func(string) (int, string, error)) *interfaceCache {
	return &interfaceCache{entries: make(map[string]interfaceInfo), names: make(map[string]struct{}), lookup: lookup}
}

func (c *interfaceCache) resolve(name string, force bool) (interfaceInfo, error) {
	if name == "" {
		return interfaceInfo{}, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.names[name] = struct{}{}
	if info, ok := c.entries[name]; ok && !force {
		return info, nil
	}
	index, ip, err := c.lookup(name)
	if err != nil {
		delete(c.entries, name)
		return interfaceInfo{}, err
	}
	info := interfaceInfo{index: index, ip: ip}
	c.entries[name] = info
	return info, nil
}

func (c *interfaceCache) invalidate() {
	c.mu.Lock()
	clear(c.entries)
	c.mu.Unlock()
}

func (c *interfaceCache) refresh() {
	c.mu.Lock()
	names := make([]string, 0, len(c.names))
	for name := range c.names {
		names = append(names, name)
	}
	c.mu.Unlock()
	for _, name := range names {
		_, _ = c.resolve(name, false)
	}
}

func (p *ProxyServer) resolveInterface(name string, force bool) (interfaceInfo, error) {
	if name == "" {
		return interfaceInfo{}, nil
	}
	if p.interfaceCache != nil {
		return p.interfaceCache.resolve(name, force)
	}
	index, ip, err := getInterfaceInfo(name)
	return interfaceInfo{index: index, ip: ip}, err
}

func isInterfaceError(err error) bool {
	for _, code := range []syscall.Errno{syscall.ENETDOWN, syscall.ENETUNREACH, syscall.EHOSTUNREACH, syscall.EADDRNOTAVAIL, syscall.ENXIO, syscall.ENODEV} {
		if errors.Is(err, code) {
			return true
		}
	}
	return isPlatformInterfaceError(err)
}

// startInterfaceMonitor runs for the application lifetime, including while SOCKS is stopped.
// It must be called before any listeners are started.
func (p *ProxyServer) startInterfaceMonitor() func() {
	cache := newInterfaceCache(getInterfaceInfo)
	changes := make(chan struct{}, 1)
	stopObserver, err := observeInterfaceChanges(func() {
		cache.invalidate()
		select {
		case changes <- struct{}{}:
		default:
		}
	})
	if err != nil {
		p.addLog("Interface notifications unavailable; resolving each connection: " + err.Error())
		return func() {}
	}
	p.interfaceCache = cache
	// Register notifications before the initial snapshot to avoid missing startup changes.
	for _, name := range []string{p.Config.DefaultIface, p.Config.GFWIface, p.Config.CompanyIface} {
		_, _ = cache.resolve(name, false)
	}
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		timer := time.NewTimer(time.Hour)
		timer.Stop()
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-changes:
				timer.Reset(100 * time.Millisecond)
			case <-timer.C:
				cache.refresh()
			}
		}
	}()
	p.addLog("Interface change notifications active")
	var once sync.Once
	return func() { once.Do(func() { stopObserver(); close(done); <-stopped }) }
}
