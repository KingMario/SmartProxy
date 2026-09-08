//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework SystemConfiguration -framework CoreFoundation
#include "interface_observer_darwin.h"
*/
import "C"

import (
	"fmt"
	"runtime/cgo"
	"sync"
)

//export goInterfaceChanged
func goInterfaceChanged(handle C.uintptr_t) {
	cgo.Handle(handle).Value().(func())()
}

func observeInterfaceChanges(changed func()) (func(), error) {
	handle := cgo.NewHandle(changed)
	observer := C.sp_interface_observer_start(C.uintptr_t(handle))
	if observer == nil {
		handle.Delete()
		return nil, fmt.Errorf("cannot subscribe to SystemConfiguration")
	}
	var once sync.Once
	return func() { once.Do(func() { C.sp_interface_observer_stop(observer); handle.Delete() }) }, nil
}
