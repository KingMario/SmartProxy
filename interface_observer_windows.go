package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	interfaceNotifyDLL           = windows.NewLazySystemDLL("iphlpapi.dll")
	notifyIPInterfaceChange      = interfaceNotifyDLL.NewProc("NotifyIpInterfaceChange")
	notifyUnicastIPAddressChange = interfaceNotifyDLL.NewProc("NotifyUnicastIpAddressChange")
	cancelMibChangeNotify        = interfaceNotifyDLL.NewProc("CancelMibChangeNotify2")
	interfaceObservers           sync.Map
	interfaceObserverID          atomic.Uintptr
	// One process-wide trampoline avoids retaining a Go closure per registration.
	interfaceNotifyCallback = windows.NewCallback(func(context, row, kind uintptr) uintptr {
		if callback, ok := interfaceObservers.Load(context); ok {
			callback.(func())()
		}
		return 0
	})
)

func observeInterfaceChanges(changed func()) (func(), error) {
	for _, proc := range []*windows.LazyProc{notifyIPInterfaceChange, notifyUnicastIPAddressChange, cancelMibChangeNotify} {
		if err := proc.Find(); err != nil {
			return nil, err
		}
	}
	id := interfaceObserverID.Add(1)
	interfaceObservers.Store(id, changed)
	var handles []windows.Handle
	var once sync.Once
	stop := func() {
		once.Do(func() {
			// Cancellation waits for callbacks. Never hold a callback's locks here.
			for _, handle := range handles {
				result, _, _ := cancelMibChangeNotify.Call(uintptr(handle))
				if result != 0 {
					log.Printf("CancelMibChangeNotify2: %v", syscall.Errno(result))
				}
			}
			interfaceObservers.Delete(id)
		})
	}
	for _, proc := range []*windows.LazyProc{notifyIPInterfaceChange, notifyUnicastIPAddressChange} {
		var handle windows.Handle
		// AF_UNSPEC subscribes to IPv4 and IPv6; initial callbacks confirm registration.
		result, _, _ := proc.Call(windows.AF_UNSPEC, interfaceNotifyCallback, id, 1, uintptr(unsafe.Pointer(&handle)))
		if result != 0 {
			stop()
			return nil, fmt.Errorf("%s: %w", proc.Name, syscall.Errno(result))
		}
		handles = append(handles, handle)
	}
	return stop, nil
}
