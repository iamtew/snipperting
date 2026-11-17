package winapp

import (
	"syscall"
	"unsafe"
)

var instanceMu uintptr

func SingleInstance(name string) bool {
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return true
	}
	k32 := syscall.NewLazyDLL("kernel32.dll")
	create := k32.NewProc("CreateMutexW")
	r, _, last := create.Call(0, 0, uintptr(unsafe.Pointer(n)))
	if r == 0 {
		return true
	}
	instanceMu = r
	const errAlreadyExists = 183
	return last != syscall.Errno(errAlreadyExists)
}
