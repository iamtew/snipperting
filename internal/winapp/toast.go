package winapp

import (
	"unsafe"
)

const (
	wsPopup        = 0x80000000
	exNoActivate   = 0x08000000
	exTopmost      = 0x00000008
	exToolWindow   = 0x00000080
	swShowNoAct    = 4
	spiGetWorkArea = 0x0030
	toastMS        = 1500
)

var (
	toastHwnd    uintptr
	procSysParam = user32.NewProc("SystemParametersInfoW")
)

func NotifyCopied() {
	if uiCfg == nil || !uiCfg.Get().ClipNotice {
		return
	}
	Open("toast", "")
}

func showCopiedToast() {
	const tw, th int32 = 240, 44
	area := workArea()
	x := area.right - tw - 16
	y := area.bottom - th - 16
	if toastHwnd == 0 {
		ex := uint32(exNoActivate | exTopmost | exToolWindow)
		st := uint32(wsPopup | wsVis | wsBorder)
		toastHwnd = createEx(ex, 0, "", "", st, x, y, tw, th, 0)
		create(toastHwnd, "STATIC", "Copied to clipboard", wsChild|wsVis, 8, 10, tw-16, 24, 0)
	} else {
		move(toastHwnd, x, y, tw, th)
	}
	pShow.Call(toastHwnd, swShowNoAct)
	procSetTimer.Call(toastHwnd, 1, toastMS, 0)
}

func hideCopiedToast() {
	if toastHwnd == 0 {
		return
	}
	destroy(toastHwnd)
	toastHwnd = 0
}

func workArea() wrect {
	var r wrect
	procSysParam.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if r.right == 0 || r.bottom == 0 {
		r.right = metrics(0)
		r.bottom = metrics(1)
	}
	return r
}
