package winapp

import (
	_ "embed"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
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
	bsOwner              = 0x000B
	tcsOwnerDraw         = 0x2000
	tvmSetBk             = 0x111D
	tvmSetText           = 0x111E
	tcmGetItem           = 0x133C
	wmCtlColorEdit       = 0x0133
	wmCtlColorBtn        = 0x0135
	wmCtlColorStatic     = 0x0138
	wmErase              = 0x0014
	wmDrawItem           = 0x002B
	wmSettingChange      = 0x001A
	odsSelected          = 0x0001
	odtButton            = 4
	odtTab               = 101
	dtCenter             = 0x0025      // DT_CENTER|DT_VCENTER|DT_SINGLELINE
	gclpBg               = ^uintptr(9) // GCLP_HBRBACKGROUND = -10
	dwmDarkMode          = 20
	dwmDarkModeOld       = 19
)

var (
	pCreateActCtx = k32.NewProc("CreateActCtxW")
	pActivateCtx  = k32.NewProc("ActivateActCtx")
	g32           = windows.NewLazySystemDLL("gdi32.dll")
	pCreateFont   = g32.NewProc("CreateFontIndirectW")
	pStockFont    = g32.NewProc("GetStockObject")
	pCreateBrush  = g32.NewProc("CreateSolidBrush")
	pDeleteObj    = g32.NewProc("DeleteObject")
	pSetBkMode    = g32.NewProc("SetBkMode")
	pSetTextColor = g32.NewProc("SetTextColor")
	pSetBkColor   = g32.NewProc("SetBkColor")
	pSelect       = g32.NewProc("SelectObject")
	pSetDPI       = u32.NewProc("SetProcessDpiAwarenessContext")
	pFillRect     = u32.NewProc("FillRect")
	pDrawText     = u32.NewProc("DrawTextW")
	pInvalidate   = u32.NewProc("InvalidateRect")
	pClassLong    = u32.NewProc("SetClassLongPtrW")
	pClassName    = u32.NewProc("GetClassNameW")
	pSetTheme     = windows.NewLazySystemDLL("uxtheme.dll").NewProc("SetWindowTheme")
	pDwmSet       = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
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

type drawItem struct {
	ctlType    uint32
	ctlID      uint32
	itemID     uint32
	itemAction uint32
	itemState  uint32
	hwndItem   uintptr
	hdc        uintptr
	rc         wrect
	itemData   uintptr
}

type palette struct {
	dark                          bool
	bg, field, text, accent       uint32
	bgBrush, fieldBrush, accBrush uintptr
	accentText                    uint32
}

var uiPal palette

func rgb(r, g, b uint8) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
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
		if ncm.messageFont.height < 0 {
			ncm.messageFont.height -= 4
		} else if ncm.messageFont.height > 0 {
			ncm.messageFont.height += 4
		}
		h, _, _ := pCreateFont.Call(uintptr(unsafe.Pointer(&ncm.messageFont)))
		if h != 0 {
			return h
		}
	}
	h, _, _ := pStockFont.Call(17)
	return h
}

func appsDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v == 0
}

func systemAccent() uint32 {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\DWM`, registry.QUERY_VALUE)
	if err != nil {
		return rgb(0, 120, 215)
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AccentColor")
	if err != nil || v == 0 {
		return rgb(0, 120, 215)
	}
	return uint32(v) & 0x00FFFFFF
}

func lum(c uint32) uint32 {
	r, g, b := c&0xff, (c>>8)&0xff, (c>>16)&0xff
	return (r*299 + g*587 + b*114) / 1000
}

func loadPalette() {
	if uiPal.bgBrush != 0 {
		pDeleteObj.Call(uiPal.bgBrush)
		pDeleteObj.Call(uiPal.fieldBrush)
		pDeleteObj.Call(uiPal.accBrush)
	}
	uiPal.dark = appsDark()
	uiPal.accent = systemAccent()
	if lum(uiPal.accent) > 160 {
		uiPal.accentText = rgb(0, 0, 0)
	} else {
		uiPal.accentText = rgb(255, 255, 255)
	}
	if uiPal.dark {
		uiPal.bg = rgb(0x20, 0x20, 0x20)
		uiPal.field = rgb(0x2b, 0x2b, 0x2b)
		uiPal.text = rgb(255, 255, 255)
	} else {
		uiPal.bg = rgb(0xf3, 0xf3, 0xf3)
		uiPal.field = rgb(255, 255, 255)
		uiPal.text = rgb(0, 0, 0)
	}
	uiPal.bgBrush, _, _ = pCreateBrush.Call(uintptr(uiPal.bg))
	uiPal.fieldBrush, _, _ = pCreateBrush.Call(uintptr(uiPal.field))
	uiPal.accBrush, _, _ = pCreateBrush.Call(uintptr(uiPal.accent))
}

func classBrush() uintptr {
	if uiPal.bgBrush != 0 {
		return uiPal.bgBrush
	}
	return 6
}

func applyFrame(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	var dark int32
	if uiPal.dark {
		dark = 1
	}
	pDwmSet.Call(hwnd, dwmDarkMode, uintptr(unsafe.Pointer(&dark)), 4)
	pDwmSet.Call(hwnd, dwmDarkModeOld, uintptr(unsafe.Pointer(&dark)), 4)
}

func setThemeName(hwnd uintptr, name string) {
	if hwnd == 0 {
		return
	}
	n, _ := windows.UTF16PtrFromString(name)
	pSetTheme.Call(hwnd, uintptr(unsafe.Pointer(n)), 0)
}

func themeExplorer(hwnd uintptr) {
	name := "Explorer"
	if uiPal.dark {
		name = "DarkMode_Explorer"
	}
	setThemeName(hwnd, name)
}

func explorerTheme(hwnd uintptr) {
	themeExplorer(hwnd)
	if hwnd == 0 {
		return
	}
	send(hwnd, tvmSetBk, 0, uintptr(uiPal.bg))
	send(hwnd, tvmSetText, 0, uintptr(uiPal.text))
}

func colorEdit(hwnd uintptr) {
	themeExplorer(hwnd)
}

func windowClass(hwnd uintptr) string {
	buf := make([]uint16, 64)
	n, _, _ := pClassName.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func ctlColor(wp, lp uintptr) uintptr {
	hdc, child := wp, lp
	cls := windowClass(child)
	bg, brush := uiPal.bg, uiPal.bgBrush
	if cls == "Edit" {
		bg, brush = uiPal.field, uiPal.fieldBrush
	}
	pSetBkMode.Call(hdc, 1)
	pSetTextColor.Call(hdc, uintptr(uiPal.text))
	pSetBkColor.Call(hdc, uintptr(bg))
	return brush
}

func eraseBg(hwnd, hdc uintptr) uintptr {
	if uiPal.bgBrush == 0 {
		return 0
	}
	var r wrect
	pClient.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), uiPal.bgBrush)
	return 1
}

func paintOwned(parent, lp uintptr) {
	di := (*drawItem)(unsafe.Pointer(lp))
	if di.ctlType == odtTab {
		paintTab(di)
		return
	}
	on := false
	if p := panels[parent]; p != nil && p.on != nil {
		on = p.on[di.hwndItem]
	}
	brush, fg := uiPal.fieldBrush, uiPal.text
	if on {
		brush, fg = uiPal.accBrush, uiPal.accentText
	}
	pFillRect.Call(di.hdc, uintptr(unsafe.Pointer(&di.rc)), brush)
	if guiFont != 0 {
		pSelect.Call(di.hdc, guiFont)
	}
	pSetBkMode.Call(di.hdc, 1)
	pSetTextColor.Call(di.hdc, uintptr(fg))
	u, _ := windows.UTF16PtrFromString(textOf(di.hwndItem))
	pDrawText.Call(di.hdc, uintptr(unsafe.Pointer(u)), ^uintptr(0), uintptr(unsafe.Pointer(&di.rc)), dtCenter)
}

func paintTab(di *drawItem) {
	sel := di.itemState&odsSelected != 0
	brush := uiPal.bgBrush
	if sel {
		brush = uiPal.fieldBrush
	}
	pFillRect.Call(di.hdc, uintptr(unsafe.Pointer(&di.rc)), brush)
	if sel {
		bar := di.rc
		bar.top = bar.bottom - 2
		pFillRect.Call(di.hdc, uintptr(unsafe.Pointer(&bar)), uiPal.accBrush)
	}
	if guiFont != 0 {
		pSelect.Call(di.hdc, guiFont)
	}
	pSetBkMode.Call(di.hdc, 1)
	pSetTextColor.Call(di.hdc, uintptr(uiPal.text))
	title := tabLabel(di.hwndItem, di.itemID)
	u, _ := windows.UTF16PtrFromString(title)
	pDrawText.Call(di.hdc, uintptr(unsafe.Pointer(u)), ^uintptr(0), uintptr(unsafe.Pointer(&di.rc)), dtCenter)
}

func tabLabel(tabs uintptr, i uint32) string {
	buf := make([]uint16, 256)
	item := tcitem{mask: tcifText, pszText: uintptr(unsafe.Pointer(&buf[0])), cchTextMax: 256}
	send(tabs, tcmGetItem, uintptr(i), uintptr(unsafe.Pointer(&item)))
	return windows.UTF16ToString(buf)
}

func invalidate(hwnd uintptr) {
	if hwnd != 0 {
		pInvalidate.Call(hwnd, 0, 1)
	}
}

func reloadTheme() {
	loadPalette()
	if uiHwnd != 0 && uiPal.bgBrush != 0 {
		pClassLong.Call(uiHwnd, gclpBg, uiPal.bgBrush)
	}
	for _, p := range panels {
		if p.hwnd == 0 {
			continue
		}
		applyFrame(p.hwnd)
		explorerTheme(p.tree)
		if p.dataEdit != 0 {
			colorEdit(p.dataEdit)
		}
		if p.clipCheck != 0 {
			themeExplorer(p.clipCheck)
		}
		invalidate(p.hwnd)
		invalidate(p.bulk)
		invalidate(p.tabs)
		invalidate(p.tree)
	}
}

func immersive(lp uintptr) bool {
	if lp == 0 {
		return true
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(lp))) == "ImmersiveColorSet"
}

func monacoDark() bool { return uiPal.dark }
