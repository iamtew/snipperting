package winapp

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procOpenClip     = user32.NewProc("OpenClipboard")
	procCloseClip    = user32.NewProc("CloseClipboard")
	procEmptyClip    = user32.NewProc("EmptyClipboard")
	procSetClip      = user32.NewProc("SetClipboardData")
	procGetClip      = user32.NewProc("GetClipboardData")
	procIsClipFormat = user32.NewProc("IsClipboardFormatAvailable")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")
)

func SetClipboard(text string) error {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	if err := openClip(); err != nil {
		return err
	}
	defer procCloseClip.Call()
	procEmptyClip.Call()
	n := len(utf16) * 2
	h, _, err := procGlobalAlloc.Call(gmemMoveable, uintptr(n))
	if h == 0 {
		return err
	}
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		return err
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(utf16))
	copy(dst, utf16)
	procGlobalUnlock.Call(h)
	r, _, err := procSetClip.Call(cfUnicodeText, h)
	if r == 0 {
		return err
	}
	return nil
}

func GetClipboard() string {
	if err := openClip(); err != nil {
		return ""
	}
	defer procCloseClip.Call()
	ok, _, _ := procIsClipFormat.Call(cfUnicodeText)
	if ok == 0 {
		return ""
	}
	h, _, _ := procGetClip.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)
	sz, _, _ := procGlobalSize.Call(h)
	if sz < 2 {
		return ""
	}
	n := int(sz / 2)
	s := unsafe.Slice((*uint16)(unsafe.Pointer(p)), n)
	if n > 0 && s[n-1] == 0 {
		n--
	}
	return syscall.UTF16ToString(s[:n])
}

func openClip() error {
	var last error
	for i := 0; i < 20; i++ {
		r, _, err := procOpenClip.Call(0)
		if r != 0 {
			return nil
		}
		last = err
		time.Sleep(10 * time.Millisecond)
	}
	return last
}
