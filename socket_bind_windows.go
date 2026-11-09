package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// Winsock ws2ipdef.h: both options use 31 at their respective protocol levels.
const (
	ipUnicastIF   = 31
	ipv6UnicastIF = 31
)

func bindSocketToInterface(fd uintptr, network string, ifIndex int) error {
	if ifIndex <= 0 || uint64(ifIndex) > uint64(^uint32(0)) {
		return fmt.Errorf("invalid interface index %d", ifIndex)
	}
	if strings.HasSuffix(network, "6") {
		return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, ipv6UnicastIF, ifIndex)
	}
	if ifIndex > 0xffffff {
		return fmt.Errorf("IPv4 interface index %d exceeds 24 bits", ifIndex)
	}
	// IP_UNICAST_IF takes a DWORD in network byte order, unlike IPV6_UNICAST_IF.
	// https://learn.microsoft.com/en-us/windows/win32/winsock/ipproto-ip-socket-options
	var index [4]byte
	binary.BigEndian.PutUint32(index[:], uint32(ifIndex))
	return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, ipUnicastIF, int(binary.NativeEndian.Uint32(index[:])))
}

// Some VPNs redirect ConnectEx to a local proxy. IP_UNICAST_IF prevents that
// loopback connection. Only permit this path when Windows routes the original
// destination through the requested interface, and the resulting socket really
// is local. Binding failures and ordinary connection failures remain errors.
func dialRedirectedVPN(ctx context.Context, target string, info interfaceInfo, timeout time.Duration, dialErr error) (net.Conn, error) {
	var syscallErr *os.SyscallError
	if info.index <= 0 || !errors.As(dialErr, &syscallErr) || !strings.EqualFold(syscallErr.Syscall, "connectex") || !errors.Is(dialErr, windows.WSAEADDRNOTAVAIL) {
		return nil, dialErr
	}
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return nil, dialErr
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, dialErr
	}
	for _, ip := range ips {
		if !windowsRouteUsesInterface(ip, info.index) {
			continue
		}
		dialer := &net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ip.String(), port))
		if err != nil {
			continue
		}
		local, ok := conn.LocalAddr().(*net.TCPAddr)
		if ok && local.IP.IsLoopback() && windowsRouteUsesInterface(ip, info.index) {
			return conn, nil
		}
		conn.Close()
	}
	return nil, dialErr
}

func windowsRouteUsesInterface(ip net.IP, index int) bool {
	ipv4 := ip.To4()
	if ipv4 == nil || ip.IsLoopback() || ip.IsUnspecified() {
		return false
	}
	var address windows.SockaddrInet4
	copy(address.Addr[:], ipv4)
	var best uint32
	return windows.GetBestInterfaceEx(&address, &best) == nil && best == uint32(index)
}
