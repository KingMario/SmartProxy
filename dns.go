package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const maxDNSCacheEntries = 1024

type dnsBinding struct {
	endpoint  string
	bootstrap string
	info      interfaceInfo
}

type dnsCacheKey struct {
	binding dnsBinding
	host    string
}

type dnsCacheEntry struct {
	ips     []string
	expires time.Time
}

type routedDNSResolver struct {
	mu         sync.Mutex
	generation uint64
	cache      map[dnsCacheKey]dnsCacheEntry
	clients    map[dnsBinding]*http.Client
	now        func() time.Time
	newClient  func(string, []string, interfaceInfo) *http.Client
}

func newRoutedDNSResolver() *routedDNSResolver {
	return &routedDNSResolver{
		cache: make(map[dnsCacheKey]dnsCacheEntry), clients: make(map[dnsBinding]*http.Client),
		now: time.Now, newClient: newBoundDNSClient,
	}
}

func (p *ProxyServer) dnsResolver() *routedDNSResolver {
	p.dnsOnce.Do(func() {
		if p.dns == nil {
			p.dns = newRoutedDNSResolver()
		}
	})
	return p.dns
}

func (p *ProxyServer) resetDNS() { p.dnsResolver().clear() }

func (r *routedDNSResolver) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generation++
	clear(r.cache)
	for _, client := range r.clients {
		client.CloseIdleConnections()
	}
	clear(r.clients)
}

func validateDNSConfig(cfg Config) error {
	for _, setting := range []struct {
		name, endpoint string
		bootstrap      []string
	}{
		{"defaultDNS", cfg.DefaultDNS, cfg.DefaultDNSBootstrap},
		{"gfwDNS", cfg.GFWDNS, cfg.GFWDNSBootstrap},
		{"companyDNS", cfg.CompanyDNS, cfg.CompanyDNSBootstrap},
	} {
		if _, err := parseDNSEndpoint(setting.endpoint); err != nil {
			return fmt.Errorf("%s: %w", setting.name, err)
		}
		for _, ip := range setting.bootstrap {
			if net.ParseIP(strings.TrimSpace(ip)) == nil {
				return fmt.Errorf("%s bootstrap must contain IP addresses", setting.name)
			}
		}
	}
	return nil
}

// Empty and "system" preserve the OS's scoped/split DNS, including company VPN DNS.
func parseDNSEndpoint(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "system") {
		return nil, nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("use system or an HTTPS DNS endpoint without credentials, query, or fragment")
	}
	if port := u.Port(); port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return nil, fmt.Errorf("invalid DNS endpoint port")
		}
	}
	return u, nil
}

func newBoundDNSClient(endpoint string, bootstrap []string, info interfaceInfo) *http.Client {
	u, _ := url.Parse(endpoint)
	transport := &http.Transport{
		// Deliberately ignore HTTP_PROXY: DNS must use this route's interface.
		Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second, DisableCompression: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if !strings.EqualFold(host, u.Hostname()) {
				return nil, fmt.Errorf("unexpected DNS service host")
			}
			if len(bootstrap) == 0 {
				return dialWithInterfaceContext(ctx, address, info)
			}
			var lastErr error
			for _, ip := range bootstrap {
				conn, err := dialWithInterfaceContext(ctx, net.JoinHostPort(strings.TrimSpace(ip), port), info)
				if err == nil {
					return conn, nil
				}
				lastErr = err
				if ctx.Err() != nil {
					break
				}
			}
			return nil, lastErr
		},
	}
	return &http.Client{Transport: transport, Timeout: 8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (r *routedDNSResolver) lookup(ctx context.Context, host string, route outboundRoute, info interfaceInfo) ([]string, error) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	u, err := parseDNSEndpoint(route.dns)
	if err != nil {
		return nil, err
	}
	if u == nil {
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		ips := make([]string, 0, len(addresses))
		for _, addr := range addresses {
			if info.ip == "" || net.ParseIP(info.ip).To4() == nil || addr.IP.To4() != nil {
				ips = append(ips, addr.IP.String())
			}
		}
		if len(ips) == 0 {
			return nil, &net.DNSError{Name: host, Err: "no address for selected interface", IsNotFound: true}
		}
		return ips, nil
	}
	binding := dnsBinding{endpoint: u.String(), bootstrap: strings.Join(route.bootstrap, ","), info: info}
	key := dnsCacheKey{binding: binding, host: host}
	r.mu.Lock()
	if entry, ok := r.cache[key]; ok && r.now().Before(entry.expires) {
		ips := append([]string(nil), entry.ips...)
		r.mu.Unlock()
		return ips, nil
	}
	generation := r.generation
	client := r.clients[binding]
	if client == nil {
		if len(r.clients) >= 16 {
			for _, old := range r.clients {
				old.CloseIdleConnections()
			}
			clear(r.clients)
		}
		client = r.newClient(binding.endpoint, route.bootstrap, info)
		r.clients[binding] = client
	}
	r.mu.Unlock()

	current := host
	visited := make(map[string]bool)
	ttl := 5 * time.Minute
	var ips []string
	for hop := 0; hop < 8; hop++ {
		if visited[current] {
			return nil, fmt.Errorf("DNS CNAME loop for %s", host)
		}
		visited[current] = true
		var next string
		var answerTTL time.Duration
		ips, next, answerTTL, err = lookupDoH(ctx, client, binding.endpoint, current)
		if err != nil {
			return nil, err
		}
		if answerTTL < ttl {
			ttl = answerTTL
		}
		if len(ips) > 0 {
			break
		}
		current = next
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("DNS CNAME limit exceeded for %s", host)
	}
	if ttl > 0 {
		r.mu.Lock()
		if generation == r.generation {
			if len(r.cache) >= maxDNSCacheEntries {
				for old, entry := range r.cache {
					if !r.now().Before(entry.expires) {
						delete(r.cache, old)
					}
				}
				if len(r.cache) >= maxDNSCacheEntries {
					for old := range r.cache {
						delete(r.cache, old)
						break
					}
				}
			}
			r.cache[key] = dnsCacheEntry{ips: append([]string(nil), ips...), expires: r.now().Add(ttl)}
		}
		r.mu.Unlock()
	}
	return ips, nil
}

func lookupDoH(ctx context.Context, client *http.Client, endpoint, host string) ([]string, string, time.Duration, error) {
	name, err := dnsmessage.NewName(host + ".")
	if err != nil {
		return nil, "", 0, err
	}
	question := dnsmessage.Question{Name: name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}
	query := dnsmessage.Message{Header: dnsmessage.Header{RecursionDesired: true}, Questions: []dnsmessage.Question{question}}
	wire, err := query.Pack()
	if err != nil {
		return nil, "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(wire))
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", 0, fmt.Errorf("DNS service returned HTTP %d", resp.StatusCode)
	}
	media, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || media != "application/dns-message" {
		return nil, "", 0, fmt.Errorf("DNS service returned an invalid content type")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, "", 0, err
	}
	if len(body) > 65535 {
		return nil, "", 0, fmt.Errorf("DNS response too large")
	}
	var answer dnsmessage.Message
	if err := answer.Unpack(body); err != nil {
		return nil, "", 0, fmt.Errorf("invalid DNS response: %w", err)
	}
	if !answer.Response || answer.ID != query.ID || answer.OpCode != 0 || answer.Truncated || len(answer.Questions) != 1 || answer.Questions[0] != question {
		return nil, "", 0, fmt.Errorf("DNS response does not match query for %s", host)
	}
	if answer.RCode != dnsmessage.RCodeSuccess {
		return nil, "", 0, &net.DNSError{Name: host, Err: answer.RCode.String(), IsNotFound: answer.RCode == dnsmessage.RCodeNameError}
	}
	current := strings.ToLower(name.String())
	ttl := 5 * time.Minute
	seen := make(map[string]bool)
	for hop := 0; hop < 8; hop++ {
		if seen[current] {
			return nil, "", 0, fmt.Errorf("DNS CNAME loop for %s", host)
		}
		seen[current] = true
		var ips []string
		var next string
		for _, resource := range answer.Answers {
			if resource.Header.Class != dnsmessage.ClassINET || !strings.EqualFold(resource.Header.Name.String(), current) {
				continue
			}
			switch data := resource.Body.(type) {
			case *dnsmessage.AResource:
				ips = append(ips, net.IP(data.A[:]).String())
			case *dnsmessage.CNAMEResource:
				next = strings.ToLower(data.CNAME.String())
			default:
				continue
			}
			if duration := time.Duration(resource.Header.TTL) * time.Second; duration < ttl {
				ttl = duration
			}
		}
		if len(ips) > 0 {
			return ips, "", ttl, nil
		}
		if next == "" {
			if current != strings.ToLower(name.String()) {
				return nil, strings.TrimSuffix(current, "."), ttl, nil
			}
			return nil, "", 0, &net.DNSError{Name: host, Err: "no IPv4 address", IsNotFound: true}
		}
		current = next
	}
	return nil, "", 0, fmt.Errorf("DNS CNAME limit exceeded for %s", host)
}
