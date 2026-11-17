package winapp

import (
	_ "embed"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed comctl.manifest
var comctlManifest []byte

const (
	actCtxProcessDefault = 0x10
	spiGetNCM            = 0x0029
	dpiPerMonitorV2      = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	exClientEdge         = 0x00000200
	tvsHasButtons        = 0x0001
	tvsShowSelAlways     = 0x0020
	tvsFullRowSelect     = 0x1000
)

var (
	pCreateActCtx = k32.NewProc("CreateActCtxW")
	pActivateCtx  = k32.NewProc("ActivateActCtx")
	g32           = windows.NewLazySystemDLL("gdi32.dll")
	pCreateFont   = g32.NewProc("CreateFontIndirectW")
	pStockFont    = g32.NewProc("GetStockObject")
	pSetDPI       = u32.NewProc("SetProcessDpiAwarenessContext")
	pSetTheme     = windows.NewLazySystemDLL("uxtheme.dll").NewProc("SetWindowTheme")
)

type actctx struct {
	size        uint32
	flags       uint32
	source      *uint16
	procArch    uint16
	langID      uint16
	assemblyDir *uint16
	resource    *uint16
	appName     *uint16
	module      uintptr
}

type logfont struct {
	height         int32
	width          int32
	escapement     int32
	orientation    int32
	weight         int32
	italic         byte
	underline      byte
	strikeOut      byte
	charSet        byte
	outPrecision   byte
	clipPrecision  byte
	quality        byte
	pitchAndFamily byte
	faceName       [32]uint16
}

type nonClientMetrics struct {
	size            uint32
	borderWidth     int32
	scrollWidth     int32
	scrollHeight    int32
	captionWidth    int32
	captionHeight   int32
	captionFont     logfont
	smCaptionWidth  int32
	smCaptionHeight int32
	smCaptionFont   logfont
	menuWidth       int32
	menuHeight      int32
	menuFont        logfont
	statusFont      logfont
	paddedBorder    int32
	messageFont     logfont
}

func enableVisualStyles() {
	f, err := os.CreateTemp("", "snipperting-comctl-*.manifest")
	if err != nil {
		return
	}
	path := f.Name()
	_, err = f.Write(comctlManifest)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(path)
		return
	}
	src, err := windows.UTF16PtrFromString(path)
	if err != nil {
		_ = os.Remove(path)
		return
	}
	ctx := actctx{size: uint32(unsafe.Sizeof(actctx{})), flags: actCtxProcessDefault, source: src}
	h, _, _ := pCreateActCtx.Call(uintptr(unsafe.Pointer(&ctx)))
	_ = os.Remove(path)
	if h == 0 || h == ^uintptr(0) {
		return
	}
	var cookie uintptr
	pActivateCtx.Call(h, uintptr(unsafe.Pointer(&cookie)))
}

func setDPI() {
	pSetDPI.Call(dpiPerMonitorV2)
}

func messageFont() uintptr {
	var ncm nonClientMetrics
	ncm.size = uint32(unsafe.Sizeof(ncm))
	ok, _, _ := procSysParam.Call(spiGetNCM, uintptr(ncm.size), uintptr(unsafe.Pointer(&ncm)), 0)
	if ok != 0 {
		h, _, _ := pCreateFont.Call(uintptr(unsafe.Pointer(&ncm.messageFont)))
		if h != 0 {
			return h
		}
	}
	h, _, _ := pStockFont.Call(17)
	return h
}

func explorerTheme(hwnd uintptr) {
	name, _ := windows.UTF16PtrFromString("Explorer")
	pSetTheme.Call(hwnd, uintptr(unsafe.Pointer(name)), 0)
}
