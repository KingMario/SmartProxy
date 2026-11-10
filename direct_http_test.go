package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func loopbackInfo(t *testing.T) interfaceInfo {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return interfaceInfo{index: iface.Index, ip: "127.0.0.1"}
		}
	}
	t.Fatal("no loopback interface")
	return interfaceInfo{}
}

func testHTTPProxy(t *testing.T, p *ProxyServer, direct bool) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if direct {
				go p.handleHTTPConnectionWithDial(conn, p.dialDirectHTTP)
			} else {
				go p.handleHTTPConnection(conn)
			}
		}
	}()
	return "http://" + listener.Addr().String()
}

func TestFixedHTTPBypassesRoutingForHTTPAndCONNECT(t *testing.T) {
	info := loopbackInfo(t)
	p := &ProxyServer{Config: Config{HTTPProxyIface: "chosen", CompanyIface: "missing-company", CompanyDomains: []string{"127.0.0.1"}, GFWProxy: "127.0.0.1:1", DefaultIface: "missing-default"}, interfaceCache: newInterfaceCache(func(name string) (int, string, error) {
		if name != "chosen" {
			t.Errorf("fixed proxy tried routing through %q", name)
			return 0, "", fmt.Errorf("unexpected interface %q", name)
		}
		return info.index, info.ip, nil
	})}
	proxyURL, _ := url.Parse(testHTTPProxy(t, p, true))
	for _, secure := range []bool{false, true} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Proxy-Authorization") != "" {
				t.Error("proxy credentials forwarded to origin")
			}
			data, _ := io.ReadAll(r.Body)
			w.Write(data)
		})
		var server *httptest.Server
		if secure {
			server = httptest.NewTLSServer(handler)
		} else {
			server = httptest.NewServer(handler)
		}
		transport := server.Client().Transport.(*http.Transport).Clone()
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.ProxyConnectHeader = http.Header{"Proxy-Authorization": []string{"Basic test"}}
		transport.DisableKeepAlives = true
		client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
		req, _ := http.NewRequest("POST", server.URL, strings.NewReader("selected-interface"))
		if !secure {
			req.Header.Set("Proxy-Authorization", "Basic test")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		transport.CloseIdleConnections()
		server.Close()
		if resp.StatusCode != 200 || string(body) != "selected-interface" {
			t.Fatalf("secure=%v: %s %s", secure, resp.Status, body)
		}
	}
}

func TestFixedHTTPMissingInterfaceReturns502(t *testing.T) {
	for _, iface := range []string{"", "smartproxy-missing-interface"} {
		p := &ProxyServer{Config: Config{HTTPProxyIface: iface}}
		proxyURL, _ := url.Parse(testHTTPProxy(t, p, true))
		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		client := &http.Client{Transport: transport, Timeout: time.Second}
		resp, err := client.Get("http://127.0.0.1:1/")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		transport.CloseIdleConnections()
		if resp.StatusCode != 502 {
			t.Fatalf("missing interface %q: %s", iface, resp.Status)
		}
	}
}

func TestHTTPProxySequentialRequestsDoNotStall(t *testing.T) {
	info := loopbackInfo(t)
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprintf("fixed=%v", direct), func(t *testing.T) {
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "response") }))
			origin.Config.IdleTimeout = time.Second
			origin.Start()
			defer origin.Close()
			p := &ProxyServer{Config: Config{HTTPProxyIface: "loopback"}, interfaceCache: newInterfaceCache(func(string) (int, string, error) { return info.index, info.ip, nil })}
			proxyURL, _ := url.Parse(testHTTPProxy(t, p, direct))
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 500 * time.Millisecond}
			for request := 0; request < 2; request++ {
				response, err := client.Get(origin.URL)
				if err != nil {
					t.Fatalf("request %d: %v", request+1, err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil || string(body) != "response" {
					t.Fatalf("request %d: body %q, err %v", request+1, body, err)
				}
			}
		})
	}
}

func TestInterfaceDNSBindsUDPAndTCPAndDoesNotFallback(t *testing.T) {
	info := loopbackInfo(t)
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	tcp, err := net.Listen("tcp4", udp.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()
	var udpQueries, tcpQueries atomic.Int32
	go func() {
		for {
			buffer := make([]byte, 4096)
			n, addr, err := udp.ReadFrom(buffer)
			if err != nil {
				return
			}
			udpQueries.Add(1)
			var query dnsmessage.Message
			if query.Unpack(buffer[:n]) != nil {
				continue
			}
			response := testDNSAnswer(&query, [4]byte{192, 0, 2, 25}, 30)
			wire, _ := response.Pack()
			udp.WriteTo(wire, addr)
		}
	}()
	go func() {
		for {
			conn, err := tcp.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				tcpQueries.Add(1)
				var size [2]byte
				if _, err := io.ReadFull(conn, size[:]); err != nil {
					return
				}
				wire := make([]byte, binary.BigEndian.Uint16(size[:]))
				if _, err := io.ReadFull(conn, wire); err != nil {
					return
				}
				var query dnsmessage.Message
				if query.Unpack(wire) != nil {
					return
				}
				answer := testDNSAnswer(&query, [4]byte{192, 0, 2, 25}, 30)
				wire, _ = answer.Pack()
				binary.BigEndian.PutUint16(size[:], uint16(len(wire)))
				conn.Write(append(size[:], wire...))
			}()
		}
	}()
	ips, err := lookupInterfaceDNS(context.Background(), "fixed-interface.invalid", info, []string{udp.LocalAddr().String()})
	if err != nil || !reflect.DeepEqual(ips, []string{"192.0.2.25"}) {
		t.Fatalf("DNS result %v %v", ips, err)
	}
	tcp.Close()
	if _, err := lookupInterfaceDNS(context.Background(), "udp-fallback.invalid", info, []string{udp.LocalAddr().String()}); err != nil {
		t.Fatal(err)
	}
	if udpQueries.Load() == 0 || tcpQueries.Load() == 0 {
		t.Fatal("DNS did not try both UDP and TCP")
	}
	if _, err := lookupInterfaceDNS(context.Background(), "www.google.com", info, nil); err == nil {
		t.Fatal("missing DNS fell back to global resolver")
	}
}

func TestScopedDNSOnlyUsesSelectedMacInterface(t *testing.T) {
	input := `DNS configuration
resolver #1
 nameserver[0] : 8.8.8.8
resolver #2
 domain : corp.example
 nameserver[0] : 10.0.0.53
 if_index : 5 (en0)
 flags : Supplemental, Request A records
DNS configuration (for scoped queries)
resolver #1
 nameserver[0] : 192.168.1.1
 nameserver[1] : 2001:db8::1
 if_index : 5 (en0)
 flags : Scoped, Request A records
resolver #2
 nameserver[0] : 1.1.1.1
 if_index : 9 (utun0)
 flags : Scoped, Request A records
resolver #3
 nameserver[0] : 192.168.1.1
 if_index : 5 (en0)
 flags : Scoped, Request A records
DNS configuration (for service-specific queries)
resolver #1
 nameserver[0] : 10.0.0.54
 if_index : 5 (en0)
 flags : Service-specific, Request A records
`
	if got := scopedDNSServers(input, 5); !reflect.DeepEqual(got, []string{"192.168.1.1"}) {
		t.Fatalf("wrong interface DNS %v", got)
	}
	if got := scopedDNSServers(input, 10); len(got) != 0 {
		t.Fatalf("unexpected global DNS fallback %v", got)
	}
	if got := scopedDNSServers("DNS configuration\nresolver #1\n nameserver[0] : 8.8.8.8\n if_index : 5 (en0)\n", 5); len(got) != 0 {
		t.Fatalf("missing scoped section used global DNS %v", got)
	}
}

func TestFixedHTTPConfigValidation(t *testing.T) {
	for _, cfg := range []Config{{Port: 17890}, {Port: 17891}, {HTTPProxyDNS: []string{"dns.example"}}, {HTTPProxyDNS: []string{"https://dns.example/dns-query"}}} {
		if validateHTTPProxyConfig(cfg) == nil {
			t.Fatalf("accepted invalid config %+v", cfg)
		}
	}
	if err := validateHTTPProxyConfig(Config{Port: 1080, HTTPProxyDNS: []string{"223.5.5.5"}}); err != nil {
		t.Fatal(err)
	}
}

func TestFixedHTTPLive(t *testing.T) {
	iface := os.Getenv("SMARTPROXY_TEST_DIRECT_IFACE")
	if iface == "" {
		t.Skip("set SMARTPROXY_TEST_DIRECT_IFACE for live DNS and proxy validation")
	}
	p := &ProxyServer{Config: Config{HTTPProxyIface: iface, VerboseLog: true,
		DefaultIface: "missing-default", CompanyIface: "missing-company",
		CompanyDomains: []string{"feishu.cn", "jd.com"}, ExtraGFWDomains: []string{"feishu.cn", "jd.com"}, GFWProxy: "127.0.0.1:1"}}
	if dns := os.Getenv("SMARTPROXY_TEST_DIRECT_DNS"); dns != "" {
		p.Config.HTTPProxyDNS = strings.Split(dns, ",")
	}
	if err := p.startDirectHTTPProxy(); err != nil {
		t.Fatal(err)
	}
	defer p.directHTTPListener.Close()
	proxyURL, _ := url.Parse("http://127.0.0.1:17891")
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, target := range []string{"http://www.feishu.cn/", "https://www.feishu.cn/", "http://read.jd.com/", "https://read.jd.com/"} {
		req, _ := http.NewRequest("HEAD", target, nil)
		response, err := client.Do(req)
		if err != nil {
			t.Errorf("%s: %v", target, err)
			continue
		}
		response.Body.Close()
		t.Logf("%s -> %s Location=%s", target, response.Status, response.Header.Get("Location"))
		if response.StatusCode == 502 {
			t.Errorf("proxy failed for %s", target)
		}
	}
}
