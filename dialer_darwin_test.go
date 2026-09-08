package main

import (
	"net"
	"syscall"
	"testing"
)

func TestDialRemoteBindsInterfaceWithoutStartupSnapshot(t *testing.T) {
	iface, err := net.InterfaceByName("lo0")
	if err != nil {
		t.Fatal(err)
	}
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
	p := &ProxyServer{Config: Config{DefaultIface: iface.Name}}
	conn, err := p.dialRemote("127.0.0.1", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	raw, err := conn.(*net.TCPConn).SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var boundIndex int
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		boundIndex, socketErr = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_IP, IP_BOUND_IF)
	}); err != nil {
		t.Fatal(err)
	}
	if socketErr != nil {
		t.Fatal(socketErr)
	}
	if boundIndex != iface.Index {
		t.Fatalf("socket bound to interface %d, want %d", boundIndex, iface.Index)
	}
}
