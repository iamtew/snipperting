package winapp

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32         = windows.NewLazySystemDLL("ole32.dll")
	shell32       = windows.NewLazySystemDLL("shell32.dll")
	pCoInit       = ole32.NewProc("CoInitializeEx")
	pCoCreate     = ole32.NewProc("CoCreateInstance")
	pSHCreateItem = shell32.NewProc("SHCreateItemFromParsingName")
)

const (
	coInitApartment = 0x2
	clsctxInProc    = 0x1
	fosPickFolders  = 0x20
	fosForceFS      = 0x40
	sigdnFileSys    = 0x80058000
)

var (
	clsidFileOpenDialog = windows.GUID{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = windows.GUID{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = windows.GUID{0x43826D1E, 0xE718, 0x42EE, [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

func hrFail(hr uintptr) bool { return int32(hr) < 0 }

func comCall(obj uintptr, idx int, args ...uintptr) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vt + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func pickFolder(owner uintptr, start string) (string, bool) {
	pCoInit.Call(0, coInitApartment)
	var dlg uintptr
	hr, _, _ := pCoCreate.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		clsctxInProc,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)),
		uintptr(unsafe.Pointer(&dlg)),
	)
	if hrFail(hr) || dlg == 0 {
		return "", false
	}
	defer comCall(dlg, 2)

	var opts uint32
	if !hrFail(comCall(dlg, 10, uintptr(unsafe.Pointer(&opts)))) {
		comCall(dlg, 9, uintptr(opts|fosPickFolders|fosForceFS))
	}
	if start != "" {
		if u, err := windows.UTF16PtrFromString(start); err == nil {
			var item uintptr
			r, _, _ := pSHCreateItem.Call(uintptr(unsafe.Pointer(u)), 0, uintptr(unsafe.Pointer(&iidIShellItem)), uintptr(unsafe.Pointer(&item)))
			if !hrFail(r) && item != 0 {
				comCall(dlg, 12, item)
				comCall(item, 2)
			}
		}
	}
	if hrFail(comCall(dlg, 3, owner)) {
		return "", false
	}
	var item uintptr
	if hrFail(comCall(dlg, 20, uintptr(unsafe.Pointer(&item)))) || item == 0 {
		return "", false
	}
	defer comCall(item, 2)
	var name *uint16
	if hrFail(comCall(item, 5, sigdnFileSys, uintptr(unsafe.Pointer(&name)))) || name == nil {
		return "", false
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(name))
	return windows.UTF16PtrToString(name), true
}
