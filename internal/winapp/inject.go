package winapp

import (
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetForeground    = user32.NewProc("GetForegroundWindow")
	procSetForeground    = user32.NewProc("SetForegroundWindow")
	procSendInput        = user32.NewProc("SendInput")
	procGetClassName     = user32.NewProc("GetClassNameW")
	procIsWindow         = user32.NewProc("IsWindow")
	procIsIconic         = user32.NewProc("IsIconic")
	procShowWindow       = user32.NewProc("ShowWindow")
	procBringToTop       = user32.NewProc("BringWindowToTop")
	procAttachThread     = user32.NewProc("AttachThreadInput")
	procGetGUIThreadInfo = user32.NewProc("GetGUIThreadInfo")
	procSetFocus         = user32.NewProc("SetFocus")
	procAllowForeground  = user32.NewProc("AllowSetForegroundWindow")
	procGetWindowThread  = user32.NewProc("GetWindowThreadProcessId")
	procGetCurrentThread = kernel32.NewProc("GetCurrentThreadId")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	lastForeground       atomic.Uintptr
	clipRestoreMu        sync.Mutex
	clipRestorePrev      string
	clipRestoreArmed     bool
)

const (
	fgWatchTimerID     = 1
	clipRestoreTimerID = 2
	clipRestoreMS      = 500
)

type guiThreadInfo struct {
	size                                         uint32
	flags                                        uint32
	hwndActive, hwndFocus, hwndCapture           uintptr
	hwndMenuOwner, hwndMoveSize, hwndCaret       uintptr
	caretLeft, caretTop, caretRight, caretBottom int32
}

func StartForegroundWatch(hwnd uintptr) {
	procSetTimer.Call(hwnd, fgWatchTimerID, 100, 0)
}

func RememberForeground() {
	h, _, _ := procGetForeground.Call()
	if keepForeground(h) {
		lastForeground.Store(h)
	}
}

func onUITimer(id uintptr) {
	if id == clipRestoreTimerID {
		flushClipRestore()
		return
	}
	RememberForeground()
}

func keepForeground(h uintptr) bool {
	if h == 0 {
		return false
	}
	var pid uint32
	procGetWindowThread.Call(h, uintptr(unsafe.Pointer(&pid)))
	if pid == uint32(os.Getpid()) {
		return false
	}
	var buf [64]uint16
	n, _, _ := procGetClassName.Call(h, uintptr(unsafe.Pointer(&buf[0])), 64)
	if n == 0 {
		return true
	}
	switch windows.UTF16ToString(buf[:]) {
	case "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow":
		return false
	}
	return true
}

func RestoreForeground() {
	h := lastForeground.Load()
	if h == 0 {
		return
	}
	if live, _, _ := procIsWindow.Call(h); live == 0 {
		return
	}
	forceForeground(h)
}

func forceForeground(hwnd uintptr) {
	procAllowForeground.Call(^uintptr(0)) // ASFW_ANY
	fg, _, _ := procGetForeground.Call()
	tgtTid, _, _ := procGetWindowThread.Call(hwnd, 0)
	fgTid, _, _ := procGetWindowThread.Call(fg, 0)
	cur, _, _ := procGetCurrentThread.Call()
	if fgTid != 0 && fgTid != cur {
		procAttachThread.Call(cur, fgTid, 1)
	}
	if tgtTid != 0 && tgtTid != cur && tgtTid != fgTid {
		procAttachThread.Call(cur, tgtTid, 1)
	}
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		procShowWindow.Call(hwnd, 9) // SW_RESTORE
	}
	procBringToTop.Call(hwnd)
	procSetForeground.Call(hwnd)
	var info guiThreadInfo
	info.size = uint32(unsafe.Sizeof(info))
	procGetGUIThreadInfo.Call(tgtTid, uintptr(unsafe.Pointer(&info)))
	focus := info.hwndCaret
	if focus == 0 {
		focus = info.hwndFocus
	}
	if focus != 0 {
		procSetFocus.Call(focus)
	}
	if tgtTid != 0 && tgtTid != cur && tgtTid != fgTid {
		procAttachThread.Call(cur, tgtTid, 0)
	}
	if fgTid != 0 && fgTid != cur {
		procAttachThread.Call(cur, fgTid, 0)
	}
}

func Inject(text string) error {
	prev := GetClipboard()
	if err := SetClipboard(text); err != nil {
		return err
	}
	RestoreForeground()
	time.Sleep(40 * time.Millisecond)
	err := sendCtrlV()
	armClipRestore(prev)
	return err
}

func armClipRestore(prev string) {
	clipRestoreMu.Lock()
	if !clipRestoreArmed {
		clipRestorePrev = prev
		clipRestoreArmed = true
	}
	clipRestoreMu.Unlock()
	h := uiHwnd
	if h == 0 {
		flushClipRestore()
		return
	}
	procSetTimer.Call(h, clipRestoreTimerID, clipRestoreMS, 0)
}

func cancelClipRestore() {
	if h := uiHwnd; h != 0 {
		procKillTimer.Call(h, clipRestoreTimerID)
	}
	clipRestoreMu.Lock()
	clipRestoreArmed = false
	clipRestorePrev = ""
	clipRestoreMu.Unlock()
}

func flushClipRestore() {
	if h := uiHwnd; h != 0 {
		procKillTimer.Call(h, clipRestoreTimerID)
	}
	clipRestoreMu.Lock()
	if !clipRestoreArmed {
		clipRestoreMu.Unlock()
		return
	}
	prev := clipRestorePrev
	clipRestoreArmed = false
	clipRestorePrev = ""
	clipRestoreMu.Unlock()
	_ = SetClipboard(prev)
}

const (
	inputKeyboard  = 1
	keyeventfKeyup = 0x0002
	vkControl      = 0x11
	vkV            = 0x56
)

// Win64 INPUT + KEYBDINPUT overlay (40 bytes).
type inputKB struct {
	Type  uint32
	_     uint32
	Vk    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	_pad  uint32
	Extra uint64
	_end  uint64
}

func sendCtrlV() error {
	inputs := []inputKB{
		{Type: inputKeyboard, Vk: vkControl},
		{Type: inputKeyboard, Vk: vkV},
		{Type: inputKeyboard, Vk: vkV, Flags: keyeventfKeyup},
		{Type: inputKeyboard, Vk: vkControl, Flags: keyeventfKeyup},
	}
	n := uintptr(len(inputs))
	r, _, err := procSendInput.Call(n, uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if r != n {
		return err
	}
	return nil
}
