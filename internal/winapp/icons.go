package winapp

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/png"
	"sync"
	"unsafe"
)

//go:embed icons/clipboard.ico
var icoClipboard []byte

//go:embed icons/keyboard.ico
var icoKeyboard []byte

//go:embed icons/folder.ico
var icoFolder []byte

//go:embed icons/pencil.ico
var icoPencil []byte

//go:embed icons/settings.ico
var icoSettings []byte

//go:embed icons/help.ico
var icoHelp []byte

//go:embed icons/info-circle.ico
var icoInfo []byte

//go:embed icons/logout.ico
var icoLogout []byte

//go:embed icons/file.ico
var icoFile []byte

//go:embed icons/file-plus.ico
var icoFilePlus []byte

//go:embed icons/folder-plus.ico
var icoFolderPlus []byte

//go:embed icons/device-floppy.ico
var icoFloppy []byte

//go:embed icons/bold.ico
var icoBold []byte

//go:embed icons/italic.ico
var icoItalic []byte

//go:embed icons/strikethrough.ico
var icoStrike []byte

//go:embed icons/heading.ico
var icoHeading []byte

//go:embed icons/quote.ico
var icoQuote []byte

//go:embed icons/list.ico
var icoList []byte

//go:embed icons/list-numbers.ico
var icoListNum []byte

//go:embed icons/code.ico
var icoCode []byte

//go:embed icons/link.ico
var icoLink []byte

//go:embed icons/markdown.ico
var icoMarkdown []byte

//go:embed icons/trash.ico
var icoTrash []byte

func menuIcon(name string) []byte {
	return chromeIcon(name)
}

func chromeIcon(name string) []byte {
	switch name {
	case "clipboard":
		return icoClipboard
	case "keyboard":
		return icoKeyboard
	case "folder":
		return icoFolder
	case "pencil":
		return icoPencil
	case "settings":
		return icoSettings
	case "help":
		return icoHelp
	case "info-circle":
		return icoInfo
	case "logout":
		return icoLogout
	case "file":
		return icoFile
	case "file-plus":
		return icoFilePlus
	case "folder-plus":
		return icoFolderPlus
	case "device-floppy":
		return icoFloppy
	case "bold":
		return icoBold
	case "italic":
		return icoItalic
	case "strikethrough":
		return icoStrike
	case "heading":
		return icoHeading
	case "quote":
		return icoQuote
	case "list":
		return icoList
	case "list-numbers":
		return icoListNum
	case "code":
		return icoCode
	case "link":
		return icoLink
	case "markdown":
		return icoMarkdown
	case "trash":
		return icoTrash
	default:
		return nil
	}
}

type bitmapinfo struct {
	size         uint32
	width        int32
	height       int32
	planes       uint16
	bitCount     uint16
	compression  uint32
	sizeImage    uint32
	xPels        int32
	yPels        int32
	clrUsed      uint32
	clrImportant uint32
}

var (
	bmpMu    sync.Mutex
	bmpCache = map[string]uintptr{}
)

func glyphBmp(name string) uintptr {
	key := name
	if uiPal.dark {
		key += "|d"
	}
	bmpMu.Lock()
	defer bmpMu.Unlock()
	if h, ok := bmpCache[key]; ok {
		return h
	}
	raw := chromeIcon(name)
	if len(raw) <= 22 {
		return 0
	}
	img, err := png.Decode(bytes.NewReader(raw[22:]))
	if err != nil {
		return 0
	}
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
	}
	if uiPal.dark {
		invertRGB(rgba)
	}
	h := dib32(rgba)
	if h != 0 {
		bmpCache[key] = h
	}
	return h
}

func invertRGB(img *image.RGBA) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		if p[i+3] == 0 {
			continue
		}
		p[i] = 255 - p[i]
		p[i+1] = 255 - p[i+1]
		p[i+2] = 255 - p[i+2]
	}
}

func dib32(img *image.RGBA) uintptr {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	bi := bitmapinfo{size: 40, width: int32(w), height: int32(-h), planes: 1, bitCount: 32}
	var bits uintptr
	hb, _, _ := pCreateDIB.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hb == 0 || bits == 0 {
		return 0
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)
	for y := 0; y < h; y++ {
		off := y * img.Stride
		for x := 0; x < w; x++ {
			i := off + x*4
			r, g, b, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
			j := (y*w + x) * 4
			dst[j+0] = byte(int(b) * int(a) / 255)
			dst[j+1] = byte(int(g) * int(a) / 255)
			dst[j+2] = byte(int(r) * int(a) / 255)
			dst[j+3] = a
		}
	}
	return hb
}

func drawGlyph(hdc uintptr, name string, rc wrect) {
	hb := glyphBmp(name)
	if hb == 0 {
		return
	}
	s := int32(iconPx)
	x := rc.left + (rc.right-rc.left-s)/2
	y := rc.top + (rc.bottom-rc.top-s)/2
	src, _, _ := pCreateDC.Call(hdc)
	if src == 0 {
		return
	}
	old, _, _ := pSelect.Call(src, hb)
	bf := uintptr(0) | uintptr(255)<<16 | uintptr(1)<<24
	pAlpha.Call(hdc, uintptr(x), uintptr(y), uintptr(s), uintptr(s), src, 0, 0, uintptr(s), uintptr(s), bf)
	pSelect.Call(src, old)
	pDeleteDC.Call(src)
}
