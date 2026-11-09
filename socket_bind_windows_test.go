package main

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsInterfaceSocketBinding(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	var index int
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			index = iface.Index
			break
		}
	}
	if index == 0 {
		t.Fatal("no loopback interface")
	}
	for _, network := range []string{"tcp4", "tcp6"} {
		t.Run(network, func(t *testing.T) {
			address, level, option := "127.0.0.1:0", windows.IPPROTO_IP, ipUnicastIF
			if network == "tcp6" {
				address, level, option = "[::1]:0", windows.IPPROTO_IPV6, ipv6UnicastIF
			}
			listener, err := net.Listen(network, address)
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
			dialer := newInterfaceDialer(interfaceInfo{index: index}, time.Second)
			conn, err := dialer.Dial(network, listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			raw, err := conn.(*net.TCPConn).SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			var bound int
			var socketErr error
			if err := raw.Control(func(fd uintptr) { bound, socketErr = windows.GetsockoptInt(windows.Handle(fd), level, option) }); err != nil {
				t.Fatal(err)
			}
			if socketErr != nil {
				t.Fatal(socketErr)
			}
			if bound != index {
				t.Fatalf("socket bound to %d, want %d", bound, index)
			}
		})
	}
}

type invalidWindowsSocket struct{}

func (invalidWindowsSocket) Control(f func(uintptr)) error {
	f(uintptr(windows.InvalidHandle))
	return nil
}
func (invalidWindowsSocket) Read(func(uintptr) bool) error  { panic("unexpected Read") }
func (invalidWindowsSocket) Write(func(uintptr) bool) error { panic("unexpected Write") }

func TestWindowsInterfaceControlPropagatesBindingError(t *testing.T) {
	err := interfaceSocketControl(9)("tcp4", "192.0.2.1:80", invalidWindowsSocket{})
	if !errors.Is(err, windows.WSAENOTSOCK) || !strings.Contains(err.Error(), "interface 9") {
		t.Fatalf("got %v, want wrapped WSAENOTSOCK with interface index", err)
	}
}

func TestWindowsInvalidInterfaceCannotFallback(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := dialWithInterfaceContext(ctx, listener.Addr().String(), interfaceInfo{index: -1, ip: "127.0.0.1"})
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("connection with invalid interface fell back to system route")
	}
	if !strings.Contains(err.Error(), "bind tcp4 socket to interface") {
		t.Fatalf("expected binding error, got %v", err)
	}
}

func TestWindowsRedirectDoesNotRetryBindingOrUnrelatedErrors(t *testing.T) {
	for _, cause := range []error{
		errors.New("socket access denied"),
		&os.SyscallError{Syscall: "setsockopt", Err: windows.WSAEADDRNOTAVAIL},
		&os.SyscallError{Syscall: "connectex", Err: windows.WSAECONNREFUSED},
		&os.SyscallError{Syscall: "connectex", Err: windows.WSAEADDRNOTAVAIL},
	} {
		// Even a ConnectEx address error must not bypass binding for loopback.
		conn, err := dialRedirectedVPN(context.Background(), "127.0.0.1:80", interfaceInfo{index: 1}, time.Second, cause)
		if conn != nil {
			conn.Close()
			t.Fatal("unexpected retry connection")
		}
		if err != cause {
			t.Fatalf("got %v, want original %v", err, cause)
		}
	}
}

// Opt in on a machine with both VPNs connected; does not modify saved settings.
func TestWindowsLiveVPNDetection(t *testing.T) {
	domain := os.Getenv("SMARTPROXY_TEST_COMPANY_DOMAIN")
	if domain == "" {
		t.Skip("set SMARTPROXY_TEST_COMPANY_DOMAIN to enable live VPN probes")
	}
	company, gfw := os.Getenv("SMARTPROXY_TEST_COMPANY_IFACE"), os.Getenv("SMARTPROXY_TEST_GFW_IFACE")
	if company == "" || gfw == "" {
		t.Fatal("set expected COMPANY_IFACE and GFW_IFACE for live probes")
	}
	p := &ProxyServer{Config: Config{CompanyDomains: []string{domain}, CompanyIface: company, GFWIface: gfw}}
	if got := p.AutoDetectCompanyIface(); got != company {
		t.Fatalf("Company interface = %q, want %q", got, company)
	}
	if got := p.AutoDetectGFWIface(); got != gfw {
		t.Fatalf("GFW interface = %q, want %q", got, gfw)
	}
	index, ip, err := getInterfaceInfo(gfw)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialInterfaceTCP(context.Background(), net.JoinHostPort(domain, "80"), interfaceInfo{index: index, ip: ip}, 3*time.Second)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatalf("GFW interface %s incorrectly reached the company server", gfw)
	}
	conn, err = p.dialRemote(domain, net.JoinHostPort(domain, "80"))
	if err != nil {
		t.Fatalf("company proxy routing: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	req, err := http.NewRequest("HEAD", "http://"+domain+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Write(conn); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 500 {
		t.Fatalf("company server returned %s", response.Status)
	}
	t.Logf("company proxy routing: %s, local=%s", response.Status, conn.LocalAddr())
}
