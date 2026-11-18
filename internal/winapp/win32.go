package winapp

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	u32         = windows.NewLazySystemDLL("user32.dll")
	pPost       = u32.NewProc("PostMessageW")
	pSend       = u32.NewProc("SendMessageW")
	pShow       = u32.NewProc("ShowWindow")
	pFocus      = u32.NewProc("SetForegroundWindow")
	pDestroy    = u32.NewProc("DestroyWindow")
	pMove       = u32.NewProc("MoveWindow")
	pClient     = u32.NewProc("GetClientRect")
	pDef        = u32.NewProc("DefWindowProcW")
	pReg        = u32.NewProc("RegisterClassExW")
	pCreate     = u32.NewProc("CreateWindowExW")
	pGetMsg     = u32.NewProc("GetMessageW")
	pPeek       = u32.NewProc("PeekMessageW")
	pTrans      = u32.NewProc("TranslateMessage")
	pDispatch   = u32.NewProc("DispatchMessageW")
	pMetrics    = u32.NewProc("GetSystemMetrics")
	pLoadImage  = u32.NewProc("LoadImageW")
	pScroll     = u32.NewProc("SetScrollInfo")
	pEnable     = u32.NewProc("EnableWindow")
	pMsgBox          = u32.NewProc("MessageBoxW")
	pCreatePopup     = u32.NewProc("CreatePopupMenu")
	pAppendMenu      = u32.NewProc("AppendMenuW")
	pTrackPopup      = u32.NewProc("TrackPopupMenu")
	pDestroyMenu     = u32.NewProc("DestroyMenu")
	pGetCursor       = u32.NewProc("GetCursorPos")
	pScreenToClient  = u32.NewProc("ScreenToClient")
	pSetCapture      = u32.NewProc("SetCapture")
	pReleaseCapture  = u32.NewProc("ReleaseCapture")
	pSetLong    = u32.NewProc("SetWindowLongPtrW")
	pCallOld    = u32.NewProc("CallWindowProcW")
	pGetDC      = u32.NewProc("GetDC")
	pReleaseDC  = u32.NewProc("ReleaseDC")
	comctl32    = windows.NewLazySystemDLL("comctl32.dll")
	pImgCreate  = comctl32.NewProc("ImageList_Create")
	pImgAdd     = comctl32.NewProc("ImageList_Add")
	pImgDestroy = comctl32.NewProc("ImageList_Destroy")
	k32         = windows.NewLazySystemDLL("kernel32.dll")
	pModule     = k32.NewProc("GetModuleHandleW")
)

type wclass struct {
	size      uint32
	style     uint32
	wndProc   uintptr
	clsExtra  int32
	wndExtra  int32
	instance  uintptr
	icon      uintptr
	cursor    uintptr
	bg        uintptr
	menu      *uint16
	className *uint16
	iconSm    uintptr
}

type wmsg struct {
	hwnd    uintptr
	message uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	pt      uint64
}

type wrect struct{ left, top, right, bottom int32 }

type wpoint struct{ x, y int32 }

func post(h, msg, wp, lp uintptr) { pPost.Call(h, msg, wp, lp) }

func pump() {
	var msg wmsg
	for {
		r, _, _ := pPeek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if r == 0 {
			return
		}
		pTrans.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func send(h, msg, wp, lp uintptr) uintptr {
	r, _, _ := pSend.Call(h, msg, wp, lp)
	return r
}

func showWin(h uintptr) { pShow.Call(h, 5) }
func focus(h uintptr)   { pFocus.Call(h) }
func destroy(h uintptr) { pDestroy.Call(h) }

func move(h uintptr, x, y, w, ht int32) {
	pMove.Call(h, uintptr(x), uintptr(y), uintptr(w), uintptr(ht), 1)
}

func clientSize(h uintptr) (int32, int32) {
	var r wrect
	pClient.Call(h, uintptr(unsafe.Pointer(&r)))
	return r.right, r.bottom
}

func defWnd(h, msg, wp, lp uintptr) uintptr {
	r, _, _ := pDef.Call(h, msg, wp, lp)
	return r
}

func metrics(i int32) int32 {
	r, _, _ := pMetrics.Call(uintptr(i))
	return int32(r)
}

func moduleHandle() uintptr {
	h, _, _ := pModule.Call(0)
	return h
}

func loadIcoSize(path string, cx, cy int32) uintptr {
	p, _ := windows.UTF16PtrFromString(path)
	h, _, _ := pLoadImage.Call(0, uintptr(unsafe.Pointer(p)), 1, uintptr(cx), uintptr(cy), 0x10)
	return h
}

func enableWin(h uintptr, on bool) {
	var v uintptr
	if on {
		v = 1
	}
	pEnable.Call(h, v)
}

func msgBox(owner uintptr, text, caption string, flags uintptr) int {
	t, _ := windows.UTF16PtrFromString(text)
	c, _ := windows.UTF16PtrFromString(caption)
	r, _, _ := pMsgBox.Call(owner, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return int(r)
}

// Alert is MB_OK|MB_ICONERROR with no owner. Used when stderr is gone (GUI subsystem).
func Alert(text string) {
	msgBox(0, text, "snipperting", 0x10)
}
