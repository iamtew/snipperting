package winapp

import (
	"time"
	"unsafe"
)

var (
	procGetForeground = user32.NewProc("GetForegroundWindow")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	procGetCursor     = user32.NewProc("GetCursorPos")
	lastForeground    uintptr
)

type point struct{ X, Y int32 }

func RememberForeground() {
	h, _, _ := procGetForeground.Call()
	lastForeground = h
}

func RestoreForeground() {
	if lastForeground != 0 {
		procSetForeground.Call(lastForeground)
	}
}

func CursorPos() (int, int) {
	var p point
	procGetCursor.Call(uintptr(unsafe.Pointer(&p)))
	return int(p.X), int(p.Y)
}

func Inject(text string) error {
	prev := GetClipboard()
	if err := SetClipboard(text); err != nil {
		return err
	}
	RestoreForeground()
	time.Sleep(40 * time.Millisecond)
	if err := sendCtrlV(); err != nil {
		_ = SetClipboard(prev)
		return err
	}
	time.Sleep(40 * time.Millisecond)
	return SetClipboard(prev)
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

var procSendInput = user32.NewProc("SendInput")

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
