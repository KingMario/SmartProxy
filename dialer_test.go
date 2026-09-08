package main

import (
	"net"
	"strings"
	"testing"
)

func TestDialRemoteHandlesInterfaceAvailabilityWithoutRestart(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	loopback := ""
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 && iface.Flags&net.FlagUp != 0 {
			loopback = iface.Name
			break
		}
	}
	if loopback == "" {
		t.Fatal("no active loopback interface")
	}

	for _, route := range []string{"company", "gfw", "default"} {
		t.Run(route, func(t *testing.T) {
			p := &ProxyServer{interfaceCache: newInterfaceCache(getInterfaceInfo), Config: Config{
				CompanyDomains:  []string{"company.test"},
				ExtraGFWDomains: []string{"gfw.test"},
			}}
			host := route + ".test"
			setInterface := func(name string) {
				switch route {
				case "company":
					p.Config.CompanyIface = name
				case "gfw":
					p.Config.GFWIface = name
				default:
					p.Config.DefaultIface = name
				}
			}
			for _, name := range []string{"sp-missing-vpn", loopback, "sp-missing-vpn", loopback, ""} {
				setInterface(name)
				conn, err := p.dialRemote(host, listener.Addr().String())
				if conn != nil {
					conn.Close()
				}
				if name == "sp-missing-vpn" {
					if err == nil || !strings.Contains(err.Error(), name) {
						t.Fatalf("missing interface: got %v, want an error naming %s", err, name)
					}
				} else if err != nil {
					t.Fatalf("interface %q without restart: %v", name, err)
				}
			}
		})
	}
}
