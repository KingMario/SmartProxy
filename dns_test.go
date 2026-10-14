package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func testDNSServer(t *testing.T, respond func(*dnsmessage.Message) *dnsmessage.Message) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/dns-message" {
			t.Errorf("unexpected DoH request: %s %s", r.Method, r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var query dnsmessage.Message
		if err := query.Unpack(body); err != nil {
			t.Error(err)
			return
		}
		answer := respond(&query)
		wire, err := answer.Pack()
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(wire)
	}))
	t.Cleanup(server.Close)
	return server
}

func testDNSAnswer(query *dnsmessage.Message, ip [4]byte, ttl uint32) *dnsmessage.Message {
	return &dnsmessage.Message{
		Header: dnsmessage.Header{ID: query.ID, Response: true}, Questions: query.Questions,
		Answers: []dnsmessage.Resource{{Header: dnsmessage.ResourceHeader{
			Name: query.Questions[0].Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: ttl,
		}, Body: &dnsmessage.AResource{A: ip}}},
	}
}

func TestDNSCacheIsScopedToExitAndExpires(t *testing.T) {
	var requests atomic.Int32
	server := testDNSServer(t, func(q *dnsmessage.Message) *dnsmessage.Message {
		requests.Add(1)
		return testDNSAnswer(q, [4]byte{192, 0, 2, 9}, 2)
	})
	r := newRoutedDNSResolver()
	now := time.Now()
	r.now = func() time.Time { return now }
	var bindings []interfaceInfo
	r.newClient = func(endpoint string, bootstrap []string, info interfaceInfo) *http.Client {
		bindings = append(bindings, info)
		return server.Client()
	}
	route := outboundRoute{dns: server.URL, bootstrap: []string{"127.0.0.1"}}
	lookup := func(info interfaceInfo) {
		t.Helper()
		ips, err := r.lookup(context.Background(), "cache.test", route, info)
		if err != nil || len(ips) != 1 || ips[0] != "192.0.2.9" {
			t.Fatalf("lookup: %v, %v", ips, err)
		}
		ips[0] = "modified"
	}
	first := interfaceInfo{index: 1, ip: "192.0.2.1"}
	second := interfaceInfo{index: 2, ip: "192.0.2.2"}
	lookup(first)
	lookup(first)
	if requests.Load() != 1 {
		t.Fatal("cache miss on same exit")
	}
	lookup(second)
	if requests.Load() != 2 || len(bindings) != 2 || bindings[1] != second {
		t.Fatal("DNS result or client shared across exits")
	}
	now = now.Add(3 * time.Second)
	lookup(first)
	if requests.Load() != 3 {
		t.Fatal("expired TTL reused")
	}
	r.clear()
	lookup(first)
	if requests.Load() != 4 {
		t.Fatal("network/config invalidation did not clear cache")
	}
	route.bootstrap = []string{"127.0.0.2"}
	lookup(first)
	if requests.Load() != 5 {
		t.Fatal("bootstrap change reused old DNS cache")
	}
}

func TestDoHFollowsCNAMEAndUsesShortestTTL(t *testing.T) {
	var requests atomic.Int32
	server := testDNSServer(t, func(q *dnsmessage.Message) *dnsmessage.Message {
		requests.Add(1)
		if q.Questions[0].Name.String() == "alias.test." {
			return &dnsmessage.Message{Header: dnsmessage.Header{Response: true}, Questions: q.Questions,
				Answers: []dnsmessage.Resource{{Header: dnsmessage.ResourceHeader{
					Name: q.Questions[0].Name, Type: dnsmessage.TypeCNAME, Class: dnsmessage.ClassINET, TTL: 1,
				}, Body: &dnsmessage.CNAMEResource{CNAME: dnsmessage.MustNewName("target.test.")}}}}
		}
		return testDNSAnswer(q, [4]byte{192, 0, 2, 8}, 30)
	})
	r := newRoutedDNSResolver()
	r.newClient = func(string, []string, interfaceInfo) *http.Client { return server.Client() }
	now := time.Now()
	r.now = func() time.Time { return now }
	route := outboundRoute{dns: server.URL}
	for attempt := 0; attempt < 2; attempt++ {
		ips, err := r.lookup(context.Background(), "alias.test", route, interfaceInfo{})
		if err != nil || len(ips) != 1 || ips[0] != "192.0.2.8" {
			t.Fatalf("CNAME lookup: %v, %v", ips, err)
		}
		now = now.Add(2 * time.Second)
	}
	if requests.Load() != 4 {
		t.Fatal("CNAME TTL did not limit cached address lifetime")
	}
}

func TestDoHRejectsInvalidAndUnrelatedResponses(t *testing.T) {
	for _, name := range []string{"wrong-id", "wrong-question", "truncated", "nxdomain", "unrelated-address", "cname-loop"} {
		t.Run(name, func(t *testing.T) {
			server := testDNSServer(t, func(q *dnsmessage.Message) *dnsmessage.Message {
				answer := testDNSAnswer(q, [4]byte{192, 0, 2, 9}, 60)
				switch name {
				case "wrong-id":
					answer.ID++
				case "wrong-question":
					answer.Questions = []dnsmessage.Question{{Name: dnsmessage.MustNewName("other.test."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}}
				case "truncated":
					answer.Truncated = true
				case "nxdomain":
					answer.RCode = dnsmessage.RCodeNameError
				case "unrelated-address":
					answer.Answers[0].Header.Name = dnsmessage.MustNewName("other.test.")
				case "cname-loop":
					answer.Answers[0].Header.Type = dnsmessage.TypeCNAME
					answer.Answers[0].Body = &dnsmessage.CNAMEResource{CNAME: q.Questions[0].Name}
				}
				return answer
			})
			if _, _, _, err := lookupDoH(context.Background(), server.Client(), server.URL, "query.test"); err == nil {
				t.Fatal("accepted invalid DNS response")
			}
		})
	}
}

func TestDNSFailureDoesNotFallBackToSystem(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer server.Close()
	r := newRoutedDNSResolver()
	r.newClient = func(string, []string, interfaceInfo) *http.Client { return server.Client() }
	_, err := r.lookup(context.Background(), "localhost", outboundRoute{dns: server.URL}, interfaceInfo{})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("failed DoH silently fell back to system DNS: %v", err)
	}
	if len(r.cache) != 0 {
		t.Fatal("cached failed lookup")
	}
}

func TestBoundDoHUsesBootstrapAndIgnoresEnvironmentProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	server := testDNSServer(t, func(q *dnsmessage.Message) *dnsmessage.Message { return testDNSAnswer(q, [4]byte{192, 0, 2, 7}, 60) })
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	// The httptest certificate covers example.com. This endpoint must use the supplied bootstrap IP.
	endpoint := "https://example.com:" + port
	client := newBoundDNSClient(endpoint, []string{"127.0.0.1"}, interfaceInfo{ip: "127.0.0.1"})
	client.Transport.(*http.Transport).TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	defer client.CloseIdleConnections()
	ips, _, _, err := lookupDoH(context.Background(), client, endpoint, "bound.test")
	if err != nil || len(ips) != 1 || ips[0] != "192.0.2.7" {
		t.Fatalf("bound bootstrap lookup: %v, %v", ips, err)
	}
}

func TestRouteDNSFollowsUserRulesAndUpstreamPriority(t *testing.T) {
	p := &ProxyServer{Config: Config{
		DefaultIface: "direct", DefaultDNS: "default", DefaultDNSBootstrap: []string{"192.0.2.1"},
		GFWIface: "vpn", GFWDNS: "gfw", GFWProxy: "upstream", CompanyIface: "company", CompanyDNS: "system",
		CompanyDomains: []string{"company.test"}, BypassDomains: []string{"bypass.test"},
		ExtraGFWDomains: []string{"company.test", "bypass.test", "blocked.test"},
	}}
	for _, test := range []struct{ host, iface, dns, upstream string }{
		{"normal.test", "direct", "default", ""}, {"192.0.2.1", "direct", "default", ""},
		{"Company.Test.", "company", "system", ""}, {"bypass.test", "direct", "default", ""},
		{"blocked.test", "vpn", "gfw", "upstream"},
	} {
		route := p.selectRoute(test.host)
		if route.iface != test.iface || route.dns != test.dns || route.upstream != test.upstream {
			t.Fatalf("%s: %+v", test.host, route)
		}
	}
	route := p.selectRoute("normal.test")
	route.bootstrap[0] = "changed"
	if p.Config.DefaultDNSBootstrap[0] != "192.0.2.1" {
		t.Fatal("route shares mutable configuration")
	}
}

func TestDialRemoteResolvesThroughSelectedDNS(t *testing.T) {
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
	server := testDNSServer(t, func(q *dnsmessage.Message) *dnsmessage.Message { return testDNSAnswer(q, [4]byte{127, 0, 0, 1}, 60) })
	p := &ProxyServer{Config: Config{DefaultDNS: server.URL}}
	p.dns = newRoutedDNSResolver()
	p.dns.newClient = func(string, []string, interfaceInfo) *http.Client { return server.Client() }
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	conn, err := p.dialRemote("unresolvable.test", net.JoinHostPort("unresolvable.test", port))
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestDNSConfigValidation(t *testing.T) {
	for _, endpoint := range []string{"", "system", " https://resolver.test/dns-query "} {
		if err := validateDNSConfig(Config{DefaultDNS: endpoint}); err != nil {
			t.Fatalf("%q: %v", endpoint, err)
		}
	}
	for _, endpoint := range []string{"http://resolver.test", "https://user:pass@resolver.test", "https://resolver.test/#fragment", "https://resolver.test/?key=value", "not a URL"} {
		if err := validateDNSConfig(Config{DefaultDNS: endpoint}); err == nil {
			t.Fatalf("accepted %q", endpoint)
		}
	}
	if err := validateDNSConfig(Config{CompanyDNSBootstrap: []string{"resolver.test"}}); err == nil {
		t.Fatal("accepted bootstrap hostname")
	}
}
