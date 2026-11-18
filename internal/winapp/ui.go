package winapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"

	"snipperting/internal/config"
	"snipperting/internal/mdhtml"
	"snipperting/internal/snippet"
)

const (
	wmOpen       = 0x8001
	wmCommand    = 0x0111
	wmClose      = 0x0010
	wmDestroy    = 0x0002
	wmSize       = 0x0005
	wmNotify     = 0x004E
	wmVScroll    = 0x0115
	wmMouseWheel = 0x020A
	wmLButton    = 0x0201
	wmSetFont    = 0x0030
	wmGetText    = 0x000D
	wmGetTextLen = 0x000E
	wmTimer      = 0x0113
	wmSetIcon    = 0x0080
	iconSmall    = 0
	iconBig      = 1

	wsOver    = 0x00CF0000
	wsVis     = 0x10000000
	wsChild   = 0x40000000
	wsPop     = 0x80000000
	wsVScroll = 0x00200000
	wsBorder  = 0x00800000
	wsTab     = 0x00010000
	wsClip    = 0x02000000

	esMulti = 0x0004
	esRead  = 0x0800
	esAutoV = 0x0040
	esAutoH = 0x0080

	bsCheck     = 0x0002
	bsAutoCheck = 0x0003
	bsPushLike  = 0x00001000
	bmGetCheck  = 0x00F0
	bmSetCheck  = 0x00F1
	bstChecked  = 1

	tviLast       = ^uintptr(0xFFFD) // (HTREEITEM)-0xFFFE
	tviRoot       = ^uintptr(0xFFFF) // (HTREEITEM)-0x10000
	tvmInsert     = 0x1132
	tvmDelete     = 0x1101
	tvmSelectItem = 0x110B
	tvgnCaret     = 9
	tvifIcon      = 0x0027       // TEXT|IMAGE|SELECTEDIMAGE|PARAM
	tvnSelChanged = ^uint32(450) // TVN_SELCHANGEDW = -451
	ilcColor32    = 0x20
	sideW         = int32(260)
	paneX         = 8 + sideW + 8
	barH          = int32(40)
	tabH          = int32(28)
	iconBtnS      = int32(32)
	iconGap       = int32(36)
	iconPx        = 24
	previewPrefix = "preview:"

	tcmInsert    = 0x133E
	tcmSetCur    = 0x130C
	tcmGetCur    = 0x130B
	tcmDelete    = 0x1308
	tcmSetItem   = 0x133D
	tcmGetCount  = 0x1304
	tcifText     = 0x0001
	wmPaint      = 0x000F
	tcnSelChange = ^uint32(550) // TCN_SELCHANGE = -551
	ttsAlways    = 0x01
	ttsNoPrefix  = 0x02
	ttfIdHwnd    = 0x0001
	ttfSubclass  = 0x0010
	ttmAddTool   = 0x0432 // TTM_ADDTOOLW
	ttmSetMaxTip = 0x0418 // TTM_SETMAXTIPWIDTH

	mbYesNoCancel = 0x00000033
	mbYesNo       = 0x00000004
	idYes         = 6
	idNo          = 7

	rowH          = 28
	emSetReadOnly = 0x00CF
)

var (
	uiOnce    sync.Once
	uiMu      sync.Mutex
	uiHwnd    uintptr
	uiQueue   []openReq
	uiCfg     *config.Store
	uiManual  string
	uiVer     string
	uiRev     string
	panels    = map[uintptr]*panel{}
	classU    []uint16
	wndCB     uintptr
	guiFont   uintptr
	appIcon   uintptr
	appIconSm uintptr
)

type openReq struct{ kind, file string }

type tnode struct {
	path string
	file bool
	item uintptr
}

type placed struct {
	hwnd uintptr
	row  int
	col  int
	x    int32
}

type tvitem struct {
	mask           uint32
	hItem          uintptr
	state          uint32
	stateMask      uint32
	pszText        uintptr
	cchTextMax     int32
	iImage         int32
	iSelectedImage int32
	cChildren      int32
	lParam         uintptr
}

type tvinsert struct {
	hParent      uintptr
	hInsertAfter uintptr
	item         tvitem
}

type nmhdr struct {
	hwndFrom uintptr
	idFrom   uintptr
	code     uint32
}

type nmtv struct {
	hdr     nmhdr
	action  uint32
	itemOld tvitem
	itemNew tvitem
	pt      uint64
}

type tcitem struct {
	mask        uint32
	dwState     uint32
	dwStateMask uint32
	pszText     uintptr
	cchTextMax  int32
	iImage      int32
	lParam      uintptr
}

type tchit struct {
	x, y  int32
	flags uint32
}

type toolinfo struct {
	size     uint32
	flags    uint32
	hwnd     uintptr
	id       uintptr
	rect     wrect
	inst     uintptr
	text     uintptr
	lParam   uintptr
	reserved uintptr
}

type tabHook struct {
	old uintptr
	p   *panel
}

var (
	tabCB    uintptr
	tabHooks = map[uintptr]tabHook{}
)

type panel struct {
	kind       string
	hwnd       uintptr
	clicks     map[uintptr]func()
	dataEdit   uintptr
	browse     uintptr
	clipCheck  uintptr
	themeSys   uintptr
	themeLight uintptr
	themeDark  uintptr
	bulk       uintptr
	placed     []placed
	rows       int
	scroll     int32
	opened     map[string]bool
	tree       uintptr
	tabs       uintptr
	host       uintptr
	nodes      []tnode
	open       []string
	dirty      map[string]bool
	on         map[uintptr]bool
	glyph      map[uintptr]string
	tip        uintptr
	tipText    map[uintptr][]uint16
	himl       uintptr
	selPath    string
	selFile    bool
	curPath    string
	pending    string
	dropAfter  string
	closing    bool
	promptOK   bool
	promptDone bool
	ready      bool
	chrome     *edge.Chromium
	dataDir    string
	btnDel     uintptr
	btnPreview uintptr
}

func StartUI(cfg *config.Store, manual, version, rev string) {
	uiOnce.Do(func() {
		uiCfg = cfg
		uiManual = manual
		uiVer = version
		uiRev = rev
		ready := make(chan struct{})
		go func() {
			runtime.LockOSThread()
			initControls()
			loadPalette()
			registerFrame()
			createPump()
			close(ready)
			messageLoop()
		}()
		<-ready
	})
}

func Open(kind, file string) {
	uiMu.Lock()
	uiQueue = append(uiQueue, openReq{kind, file})
	h := uiHwnd
	uiMu.Unlock()
	if h != 0 {
		post(uintptr(h), wmOpen, 0, 0)
	}
}

func initControls() {
	enableVisualStyles()
	setDPI()
	guiFont = messageFont()
	var icc struct{ size, flags uint32 }
	icc.size = 8
	icc.flags = 0xFF
	windows.NewLazySystemDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&icc)))
}

func registerFrame() {
	classU, _ = windows.UTF16FromString("snippertingFrame")
	wndCB = windows.NewCallback(wndProc)
	if ico := findIco(); ico != "" {
		appIcon = loadIcoSize(ico, metrics(11), metrics(12))
		appIconSm = loadIcoSize(ico, metrics(49), metrics(50))
	}
	wc := wclass{
		size:      uint32(unsafe.Sizeof(wclass{})),
		wndProc:   wndCB,
		instance:  moduleHandle(),
		icon:      appIcon,
		iconSm:    appIconSm,
		bg:        classBrush(),
		className: &classU[0],
	}
	pReg.Call(uintptr(unsafe.Pointer(&wc)))
}

func createPump() {
	h := create(0, "", "", 0, 0, 0, 0, 0, 0)
	uiMu.Lock()
	uiHwnd = uintptr(h)
	uiMu.Unlock()
	StartForegroundWatch(uiHwnd)
	post(uintptr(h), wmOpen, 0, 0)
}

func messageLoop() {
	var msg wmsg
	for {
		r, _, _ := pGetMsg.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 || int32(r) == -1 {
			return
		}
		pTrans.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	if msg == wmSettingChange && immersive(lp) {
		reloadTheme()
		return 0
	}
	if msg == wmCtlColorStatic || msg == wmCtlColorEdit || msg == wmCtlColorBtn {
		if b := ctlColor(wp, lp); b != 0 {
			return b
		}
	}
	if msg == wmErase {
		return eraseBg(hwnd, wp)
	}
	if msg == wmDrawItem {
		paintOwned(hwnd, lp)
		return 1
	}
	if msg == wmOpen {
		drain()
		return 0
	}
	if msg == wmTimer {
		if hwnd == toastHwnd {
			hideCopiedToast()
			return 0
		}
		WatchForeground()
		return 0
	}
	if p := panels[hwnd]; p != nil {
		if p.onMsg(hwnd, msg, wp, lp) {
			return 0
		}
	}
	return defWnd(hwnd, msg, wp, lp)
}

func drain() {
	uiMu.Lock()
	q := uiQueue
	uiQueue = nil
	uiMu.Unlock()
	for _, r := range q {
		show(r.kind, r.file)
	}
}

func show(kind, file string) {
	if kind == "toast" {
		showCopiedToast()
		return
	}
	for _, p := range panels {
		if p.kind == kind && p.hwnd != 0 {
			showWin(p.hwnd)
			focus(p.hwnd)
			if kind == "prefs" {
				p.rebuildPrefs()
			}
			if kind == "editor" {
				p.dataDir = uiCfg.Get().DataDir
				p.fillTree()
				if file != "" {
					p.loadFile(file)
				}
			}
			return
		}
	}
	switch kind {
	case "prefs":
		openPrefs()
	case "help":
		openHelp()
	case "about":
		openAbout()
	case "editor":
		openEditor(file)
	}
}

func (p *panel) onMsg(hwnd, msg, wp, lp uintptr) bool {
	switch msg {
	case wmCommand:
		if fn := p.clicks[lp]; fn != nil {
			fn()
			return true
		}
	case wmNotify:
		if p.kind == "editor" {
			p.onNotify(lp)
			return true
		}
	case wmSize:
		if hwnd == p.hwnd {
			p.onSize()
		}
	case wmVScroll:
		if hwnd == p.bulk {
			p.onScroll(wp)
			return true
		}
	case wmMouseWheel:
		if p.bulk != 0 {
			p.scroll -= int32(int16(wp>>16)) / 4
			p.layoutBulk()
			return true
		}
	case wmClose:
		if hwnd == p.hwnd {
			if p.kind == "editor" && !p.canClose() {
				return true
			}
			if p.kind == "prompt" {
				p.promptDone = true
			}
			destroy(hwnd)
			return true
		}
	case wmDestroy:
		if hwnd == p.hwnd {
			delete(panels, hwnd)
			if p.bulk != 0 {
				delete(panels, p.bulk)
			}
			if p.tabs != 0 {
				delete(tabHooks, p.tabs)
			}
			if p.himl != 0 {
				pImgDestroy.Call(p.himl)
				p.himl = 0
			}
			p.hwnd = 0
			if p.kind == "prompt" {
				p.promptDone = true
			}
		}
	}
	return false
}

func openPrefs() {
	p := &panel{kind: "prefs", clicks: map[uintptr]func(){}, opened: map[string]bool{}, on: map[uintptr]bool{}}
	p.hwnd = uintptr(frame("Preferences", 720, 560))
	panels[p.hwnd] = p
	p.static(p.hwnd, "Data directory", 12, 12, 200, 18)
	p.dataEdit = p.edit(p.hwnd, "", 12, 32, 580, 24, false)
	send(p.dataEdit, emSetReadOnly, 1, 0)
	p.browse = p.button(p.hwnd, "Browse", 600, 30, 80, 26)
	p.clicks[p.browse] = p.pickDataDir
	p.clipCheck = uintptr(create(p.hwnd, "BUTTON", "Show copied notice", wsChild|wsVis|wsTab|bsAutoCheck, 12, 64, 220, 24, 0))
	themeExplorer(p.clipCheck)
	p.clicks[p.clipCheck] = p.saveClipNotice
	p.static(p.hwnd, "Theme", 12, 94, 80, 18)
	p.themeSys = p.pushLike(p.hwnd, "System", false)
	p.themeLight = p.pushLike(p.hwnd, "Light", false)
	p.themeDark = p.pushLike(p.hwnd, "Dark", false)
	move(p.themeSys, 12, 114, 80, 24)
	move(p.themeLight, 96, 114, 80, 24)
	move(p.themeDark, 180, 114, 80, 24)
	p.clicks[p.themeSys] = func() { p.applyTheme("system") }
	p.clicks[p.themeLight] = func() { p.applyTheme("light") }
	p.clicks[p.themeDark] = func() { p.applyTheme("dark") }
	p.static(p.hwnd, "Bulk Edit", 12, 148, 200, 18)
	p.bulk = uintptr(create(p.hwnd, "", "", wsChild|wsVis|wsVScroll|wsClip, 0, 170, 704, 350, 0))
	panels[p.bulk] = p
	p.rebuildPrefs()
}

func (p *panel) clipNotice() bool {
	return send(p.clipCheck, bmGetCheck, 0, 0) != 0
}

func (p *panel) pickDataDir() {
	dir, ok := pickFolder(p.hwnd, textOf(p.dataEdit))
	if !ok {
		return
	}
	cur := uiCfg.Get()
	_ = uiCfg.Set(config.File{DataDir: dir, Theme: cur.Theme, ClipNotice: p.clipNotice()})
	p.rebuildPrefs()
}

func (p *panel) saveClipNotice() {
	cur := uiCfg.Get()
	_ = uiCfg.Set(config.File{DataDir: cur.DataDir, Theme: cur.Theme, ClipNotice: p.clipNotice()})
}

func (p *panel) applyTheme(name string) {
	cur := uiCfg.Get()
	_ = uiCfg.Set(config.File{DataDir: cur.DataDir, Theme: name, ClipNotice: p.clipNotice()})
	p.markTheme(name)
	reloadTheme()
}

func (p *panel) markTheme(name string) {
	if p.on == nil {
		p.on = map[uintptr]bool{}
	}
	p.on[p.themeSys] = name == "system"
	p.on[p.themeLight] = name == "light"
	p.on[p.themeDark] = name == "dark"
	invalidate(p.themeSys)
	invalidate(p.themeLight)
	invalidate(p.themeDark)
}

func (p *panel) rebuildPrefs() {
	for _, pl := range p.placed {
		destroy(pl.hwnd)
		delete(p.clicks, pl.hwnd)
		delete(p.on, pl.hwnd)
	}
	p.placed = nil
	p.rows = 0
	p.scroll = 0
	cur := uiCfg.Get()
	setText(p.dataEdit, cur.DataDir)
	p.markTheme(cur.Theme)
	if p.clipCheck != 0 {
		v := uintptr(0)
		if cur.ClipNotice {
			v = bstChecked
		}
		send(p.clipCheck, bmSetCheck, v, 0)
	}
	tree, err := snippet.Walk(cur.DataDir)
	if err == nil && len(tree.Children) > 0 {
		snippet.OrderChildren(tree.Children, false)
		p.addBulk(tree.Children, 0)
	} else {
		lab := p.static(p.bulk, "No snippets yet. Create notes in Edit.", 0, 0, 400, 24)
		p.placed = append(p.placed, placed{hwnd: lab, row: 0, col: 0, x: 12})
		p.rows = 1
	}
	p.layoutBulk()
}

func (p *panel) addBulk(nodes []snippet.Node, depth int) {
	for _, n := range nodes {
		n := n
		row := p.rows
		p.rows++
		x := int32(12 + depth*16)
		if n.Type == "dir" {
			title := "▸ " + n.Name
			if p.opened[n.Path] {
				title = "▾ " + n.Name
			}
			b := p.button(p.bulk, title, 0, 0, 200, 24)
			p.placed = append(p.placed, placed{hwnd: b, row: row, col: 0, x: x})
			path := n.Path
			p.clicks[b] = func() {
				p.opened[path] = !p.opened[path]
				p.rebuildPrefs()
			}
			if p.opened[n.Path] {
				p.addBulk(n.Children, depth+1)
			}
			continue
		}
		lab := p.static(p.bulk, n.Name, 0, 0, 200, 24)
		p.placed = append(p.placed, placed{hwnd: lab, row: row, col: 0, x: x})
		act := n.Action
		if act != "inject" {
			act = "clipboard"
		}
		st := "off"
		if n.Enabled {
			st = "on"
		}
		hid := "shown"
		if n.Hidden {
			hid = "hidden"
		}
		path := n.Path
		inj := p.pushLike(p.bulk, "Inject", act == "inject")
		clip := p.pushLike(p.bulk, "Clipboard", act == "clipboard")
		sb := p.button(p.bulk, st, 0, 0, 64, 24)
		hb := p.button(p.bulk, hid, 0, 0, 72, 24)
		p.placed = append(p.placed,
			placed{hwnd: inj, row: row, col: 1},
			placed{hwnd: clip, row: row, col: 2},
			placed{hwnd: sb, row: row, col: 3},
			placed{hwnd: hb, row: row, col: 4},
		)
		p.clicks[inj] = func() {
			_ = snippet.UpdateMeta(uiCfg.Get().DataDir, path, func(m *snippet.Meta) { m.Action = "inject" })
			p.rebuildPrefs()
		}
		p.clicks[clip] = func() {
			_ = snippet.UpdateMeta(uiCfg.Get().DataDir, path, func(m *snippet.Meta) { m.Action = "clipboard" })
			p.rebuildPrefs()
		}
		p.clicks[sb] = func() {
			_ = snippet.UpdateMeta(uiCfg.Get().DataDir, path, func(m *snippet.Meta) { m.Enabled = !n.Enabled })
			p.rebuildPrefs()
		}
		p.clicks[hb] = func() {
			_ = snippet.UpdateMeta(uiCfg.Get().DataDir, path, func(m *snippet.Meta) { m.Hidden = !n.Hidden })
			p.rebuildPrefs()
		}
	}
}

func (p *panel) layoutBulk() {
	if p.bulk == 0 {
		return
	}
	w, bh := clientSize(p.bulk)
	if w < 200 {
		w = 200
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
	maxScroll := int32(p.rows*rowH) - bh
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scroll > maxScroll {
		p.scroll = maxScroll
	}
	for _, pl := range p.placed {
		y := int32(pl.row*rowH) - p.scroll
		switch pl.col {
		case 0:
			move(pl.hwnd, pl.x, y, w-pl.x-316, rowH-4)
		case 1:
			move(pl.hwnd, w-308, y, 80, rowH-4)
		case 2:
			move(pl.hwnd, w-224, y, 80, rowH-4)
		case 3:
			move(pl.hwnd, w-140, y, 64, rowH-4)
		case 4:
			move(pl.hwnd, w-72, y, 72, rowH-4)
		}
	}
	setScroll(p.bulk, int32(p.rows*rowH), uint32(bh), p.scroll)
}

func (p *panel) onScroll(wp uintptr) {
	code := uint16(wp)
	switch code {
	case 0:
		p.scroll -= rowH
	case 1:
		p.scroll += rowH
	case 2:
		p.scroll -= 120
	case 3:
		p.scroll += 120
	case 5, 4:
		p.scroll = int32(wp >> 16)
	}
	p.layoutBulk()
}

func openHelp() {
	p := &panel{kind: "help", clicks: map[uintptr]func(){}}
	p.hwnd = uintptr(frame("Help", 720, 640))
	panels[p.hwnd] = p
	p.host = uintptr(create(p.hwnd, "", "", wsChild|wsVis|wsClip, 8, 8, 704, 624, 0))
	p.chrome = edge.NewChromium()
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		p.chrome.DataPath = filepath.Join(local, "snipperting", "webview")
	}
	if !p.chrome.Embed(p.host) {
		p.chrome = nil
		ed := p.edit(p.hwnd, "", 8, 8, 688, 584, true)
		setText(ed, uiManual)
		send(ed, 0x00C5, 0x7ffffffe, 0)
		p.dataEdit = ed
		return
	}
	p.chrome.Resize()
	p.ready = true
	p.showHelp()
}

func (p *panel) showHelp() {
	if p.chrome == nil {
		return
	}
	p.chrome.NavigateToString(mdhtml.Page("Help", uiManual, monacoDark()))
}

func openAbout() {
	p := &panel{kind: "about", clicks: map[uintptr]func(){}}
	p.hwnd = uintptr(frame("About", 420, 280))
	panels[p.hwnd] = p
	if ico := findIco(); ico != "" {
		if h := loadIco(ico); h != 0 {
			st := uintptr(create(p.hwnd, "STATIC", "", wsChild|wsVis|0x0003, 24, 24, 64, 64, 0))
			send(st, 0x0172, 1, h)
		}
	}
	p.static(p.hwnd, "snipperting", 104, 28, 280, 24)
	p.static(p.hwnd, "v"+uiVer, 104, 52, 280, 20)
	if uiRev != "" {
		p.static(p.hwnd, uiRev, 104, 74, 280, 20)
	}
	p.static(p.hwnd, "Snippets from the Windows tray.", 24, 110, 360, 40)
}

func openEditor(file string) {
	p := &panel{kind: "editor", clicks: map[uintptr]func(){}, dataDir: uiCfg.Get().DataDir, dirty: map[string]bool{}, on: map[uintptr]bool{}, glyph: map[uintptr]string{}, tipText: map[uintptr][]uint16{}}
	p.hwnd = uintptr(frame("Editor", 1100, 720))
	panels[p.hwnd] = p
	p.tip = uintptr(create(p.hwnd, "tooltips_class32", "", wsPop|ttsAlways|ttsNoPrefix, 0, 0, 0, 0, 0))
	send(p.tip, ttmSetMaxTip, 0, 400)
	note := p.iconBtn(p.hwnd, "New Note", "file-plus", 8, 4)
	folder := p.iconBtn(p.hwnd, "New Folder", "folder-plus", 8+iconGap, 4)
	save := p.iconBtn(p.hwnd, "Save", "device-floppy", 8+iconGap*2, 4)
	p.btnDel = p.iconBtn(p.hwnd, "Delete", "trash", 8, 4)
	p.clicks[note] = p.newNote
	p.clicks[folder] = p.newFolder
	p.clicks[save] = p.saveActive
	p.clicks[p.btnDel] = p.deleteSel
	x := paneX
	for _, f := range []struct{ title, icon, kind string }{
		{"Bold", "bold", "bold"},
		{"Italic", "italic", "italic"},
		{"Strikethrough", "strikethrough", "strike"},
		{"Heading", "heading", "heading"},
		{"Quote", "quote", "quote"},
		{"List", "list", "list"},
		{"Numbered list", "list-numbers", "olist"},
		{"Code", "code", "code"},
		{"Link", "link", "link"},
	} {
		b := p.iconBtn(p.hwnd, f.title, f.icon, x, 4)
		kind := f.kind
		p.clicks[b] = func() { p.snipFmt(kind) }
		x += iconGap
	}
	p.btnPreview = p.iconBtn(p.hwnd, "Preview", "markdown", x, 4)
	p.clicks[p.btnPreview] = p.openPreview
	p.layoutBar()
	p.tree = uintptr(create(p.hwnd, "SysTreeView32", "", wsChild|wsVis|tvsHasButtons|tvsShowSelAlways|tvsFullRowSelect, 8, barH, sideW, 640, 0))
	explorerTheme(p.tree)
	p.treeIcons()
	p.tabs = uintptr(create(p.hwnd, "SysTabControl32", "", wsChild|wsVis|wsClip|tcsOwnerDraw|tcsRagged, paneX, barH, 800, tabH, 0))
	clearTheme(p.tabs)
	send(p.tabs, tcmSetPadding, 0, uintptr(14|(4<<16)))
	hookTabs(p)
	p.host = uintptr(create(p.hwnd, "", "", wsChild|wsVis|wsClip, paneX, barH+tabH, 800, 612, 0))
	p.fillTree()
	p.chrome = edge.NewChromium()
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		p.chrome.DataPath = filepath.Join(local, "snipperting", "webview")
	}
	p.chrome.MessageCallback = func(msg string) {
		var m struct {
			T     string `json:"t"`
			V     string `json:"v"`
			P     string `json:"p"`
			Files []struct {
				P string `json:"p"`
				V string `json:"v"`
			} `json:"files"`
		}
		if json.Unmarshal([]byte(msg), &m) != nil {
			return
		}
		if m.T == "ready" {
			p.ready = true
			p.push()
			return
		}
		if m.T == "preview" {
			p.showPreviewHTML(snippet.Parse([]byte(m.V)).Body)
			return
		}
		if m.T == "dirty" && m.P != "" {
			p.dirty[m.P] = true
			p.refreshTab(m.P)
			return
		}
		if m.T == "saveall" {
			for _, f := range m.Files {
				if f.P != "" && !isPreview(f.P) {
					_ = snippet.Write(p.dataDir, f.P, []byte(f.V))
					p.dirty[f.P] = false
					p.refreshTab(f.P)
				}
			}
			if p.closing {
				destroy(p.hwnd)
			}
			return
		}
		if m.T == "save" {
			rel := m.P
			if rel == "" {
				rel = p.curPath
			}
			if rel != "" && !isPreview(rel) {
				_ = snippet.Write(p.dataDir, rel, []byte(m.V))
				p.dirty[rel] = false
				p.refreshTab(rel)
			}
			if p.dropAfter != "" && p.dropAfter == rel {
				p.dropAfter = ""
				p.removeTab(rel)
			}
			if p.closing {
				destroy(p.hwnd)
			}
		}
	}
	if !p.chrome.Embed(p.host) {
		p.webViewFail()
		return
	}
	root := findWebRoot()
	wv3 := p.chrome.GetICoreWebView2_3()
	if root == "" || wv3 == nil {
		p.webViewFail()
		return
	}
	if err := wv3.SetVirtualHostNameToFolderMapping(editorHost, root, edge.COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_ALLOW); err != nil {
		p.webViewFail()
		return
	}
	p.chrome.Resize()
	p.chrome.Navigate(fmt.Sprintf("https://%s/editor.html?v=%s", editorHost, uiVer))
	if file != "" {
		p.loadFile(file)
	}
}

func (p *panel) webViewFail() {
	if p.chrome != nil {
		_ = p.chrome.Hide()
	}
	p.static(p.host, "WebView2 failed to start.", 8, 8, 400, 24)
	p.chrome = nil
}

func (p *panel) fillTree() {
	if p.tree == 0 {
		return
	}
	keep := p.selPath
	send(p.tree, tvmDelete, 0, tviRoot)
	p.nodes = nil
	tree, err := snippet.Walk(p.dataDir)
	if err != nil {
		return
	}
	snippet.OrderChildren(tree.Children, true)
	p.addTree(tviRoot, tree.Children)
	p.selectTree(keep)
}

func (p *panel) addTree(parent uintptr, nodes []snippet.Node) {
	for _, n := range nodes {
		img := int32(0)
		if n.Type == "file" {
			img = 1
		}
		idx := len(p.nodes)
		h := p.insertTree(parent, n.Name, uintptr(idx), img)
		p.nodes = append(p.nodes, tnode{path: n.Path, file: n.Type == "file", item: h})
		if n.Type == "dir" {
			p.addTree(h, n.Children)
		}
	}
}

func (p *panel) insertTree(parent uintptr, label string, idx uintptr, img int32) uintptr {
	u, _ := windows.UTF16FromString(label)
	ins := tvinsert{
		hParent:      parent,
		hInsertAfter: tviLast,
		item: tvitem{
			mask:           tvifIcon,
			pszText:        uintptr(unsafe.Pointer(&u[0])),
			iImage:         img,
			iSelectedImage: img,
			lParam:         idx,
		},
	}
	return send(p.tree, tvmInsert, 0, uintptr(unsafe.Pointer(&ins)))
}

func (p *panel) treeIcons() {
	if p.tree == 0 {
		return
	}
	if p.himl != 0 {
		pImgDestroy.Call(p.himl)
		p.himl = 0
	}
	h, _, _ := pImgCreate.Call(uintptr(iconPx), uintptr(iconPx), ilcColor32, 2, 1)
	if h == 0 {
		return
	}
	p.himl = h
	pImgAdd.Call(h, glyphBmp("folder"), 0)
	pImgAdd.Call(h, glyphBmp("file"), 0)
	send(p.tree, tvmSetImageList, 0, h)
	send(p.tree, tvmSetIndent, 19, 0)
}

func (p *panel) iconBtn(parent uintptr, title, name string, x, y int32) uintptr {
	h := p.button(parent, title, x, y, iconBtnS, iconBtnS)
	p.glyph[h] = name
	p.addTip(h, title)
	return h
}

func (p *panel) addTip(btn uintptr, title string) {
	if p.tip == 0 {
		return
	}
	// trailing space: tooltip width is often one glyph short, which would clip the last letter
	u, _ := windows.UTF16FromString(title + " ")
	p.tipText[btn] = u
	ti := toolinfo{
		size:  uint32(unsafe.Sizeof(toolinfo{})),
		flags: ttfIdHwnd | ttfSubclass,
		hwnd:  p.hwnd,
		id:    btn,
		text:  uintptr(unsafe.Pointer(&p.tipText[btn][0])),
	}
	send(p.tip, ttmAddTool, 0, uintptr(unsafe.Pointer(&ti)))
}

func (p *panel) snipFmt(kind string) {
	if p.chrome == nil || !p.ready {
		return
	}
	k, _ := json.Marshal(kind)
	p.chrome.Eval("window.snipFmt&&window.snipFmt(" + string(k) + ")&&true")
}

func hookTabs(p *panel) {
	if p.tabs == 0 {
		return
	}
	if tabCB == 0 {
		tabCB = windows.NewCallback(tabWndProc)
	}
	old, _, _ := pSetLong.Call(p.tabs, ^uintptr(3), tabCB) // GWLP_WNDPROC = -4
	tabHooks[p.tabs] = tabHook{old: old, p: p}
}

func tabWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	h := tabHooks[hwnd]
	if msg == wmErase {
		return eraseBg(hwnd, wp)
	}
	if msg == wmLButton && h.p != nil && h.p.tabCloseHit(lp) {
		return 0
	}
	var r uintptr
	if h.old != 0 {
		r, _, _ = pCallOld.Call(h.old, hwnd, msg, wp, lp)
	} else {
		r = defWnd(hwnd, msg, wp, lp)
	}
	if msg == wmPaint {
		fillTabRest(hwnd)
	}
	return r
}

func fillTabRest(hwnd uintptr) {
	if uiPal.bgBrush == 0 {
		return
	}
	hdc, _, _ := pGetDC.Call(hwnd)
	if hdc == 0 {
		return
	}
	n := int(send(hwnd, tcmGetCount, 0, 0))
	for i := 0; i < n; i++ {
		var ir wrect
		send(hwnd, tcmGetItemRect, uintptr(i), uintptr(unsafe.Pointer(&ir)))
		pExcludeClip.Call(hdc, uintptr(ir.left), uintptr(ir.top), uintptr(ir.right), uintptr(ir.bottom))
	}
	var cr wrect
	pClient.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&cr)), uiPal.bgBrush)
	pReleaseDC.Call(hwnd, hdc)
}

func (p *panel) tabCloseHit(lp uintptr) bool {
	hit := tchit{x: int32(int16(lp)), y: int32(int16(lp >> 16))}
	i := int(send(p.tabs, tcmHitTest, 0, uintptr(unsafe.Pointer(&hit))))
	if i < 0 || i >= len(p.open) {
		return false
	}
	var rc wrect
	if send(p.tabs, tcmGetItemRect, uintptr(i), uintptr(unsafe.Pointer(&rc))) == 0 {
		return false
	}
	if hit.x < rc.right-18 {
		return false
	}
	p.closePath(p.open[i])
	return true
}

func (p *panel) onNotify(lp uintptr) {
	hdr := (*nmhdr)(unsafe.Pointer(lp))
	if hdr.hwndFrom == p.tabs && hdr.code == tcnSelChange {
		i := int(send(p.tabs, tcmGetCur, 0, 0))
		p.activate(i)
		return
	}
	if hdr.hwndFrom != p.tree {
		return
	}
	nm := (*nmtv)(unsafe.Pointer(lp))
	if nm.hdr.code != tvnSelChanged {
		return
	}
	i := int(nm.itemNew.lParam)
	if i < 0 || i >= len(p.nodes) {
		return
	}
	n := p.nodes[i]
	p.selPath = n.path
	p.selFile = n.file
	if n.file {
		p.openPath(n.path)
	}
}

func (p *panel) selectTree(rel string) {
	if rel == "" || p.tree == 0 {
		return
	}
	for _, n := range p.nodes {
		if n.path != rel {
			continue
		}
		p.selPath = n.path
		p.selFile = n.file
		send(p.tree, tvmSelectItem, tvgnCaret, n.item)
		return
	}
}

func (p *panel) loadFile(rel string) {
	p.openPath(rel)
}

func isPreview(rel string) bool {
	return strings.HasPrefix(rel, previewPrefix)
}

func previewOf(rel string) string {
	return previewPrefix + rel
}

func previewSource(rel string) string {
	return strings.TrimPrefix(rel, previewPrefix)
}

func (p *panel) openPreview() {
	src := p.curPath
	if src == "" {
		return
	}
	if isPreview(src) {
		p.refreshPreview()
		return
	}
	prev := previewOf(src)
	for i, t := range p.open {
		if t == prev {
			send(p.tabs, tcmSetCur, uintptr(i), 0)
			p.activate(i)
			return
		}
	}
	p.open = append(p.open, prev)
	p.insertTab(len(p.open)-1, tabTitle(prev, false))
	send(p.tabs, tcmSetCur, uintptr(len(p.open)-1), 0)
	p.activate(len(p.open) - 1)
}

func (p *panel) refreshPreview() {
	if p.chrome == nil || !p.ready || !isPreview(p.curPath) {
		return
	}
	src := previewSource(p.curPath)
	for _, t := range p.open {
		if t == src {
			path, _ := json.Marshal(src)
			p.chrome.Eval(fmt.Sprintf("window.snipPreviewAsk&&window.snipPreviewAsk(%s)", string(path)))
			return
		}
	}
	raw, err := snippet.Read(p.dataDir, src)
	if err != nil {
		return
	}
	p.showPreviewHTML(snippet.Parse(raw).Body)
}

func (p *panel) showPreviewHTML(body string) {
	if p.chrome == nil || !p.ready {
		return
	}
	h, _ := json.Marshal(mdhtml.HTML(body))
	p.chrome.Eval(fmt.Sprintf("window.snipPreview&&window.snipPreview(%s,%t)", string(h), monacoDark()))
}

func (p *panel) push() {
	if p.chrome == nil || !p.ready {
		return
	}
	if isPreview(p.curPath) {
		p.refreshPreview()
		return
	}
	b, _ := json.Marshal(p.pending)
	dark := monacoDark()
	path, _ := json.Marshal(p.curPath)
	if p.curPath != "" {
		p.chrome.Eval(fmt.Sprintf("window.snipOpen&&window.snipOpen(%s,%s,%t)", string(path), string(b), dark))
		return
	}
	p.idlePane()
}

func (p *panel) idlePane() {
	if p.chrome == nil || !p.ready {
		return
	}
	p.chrome.Eval(fmt.Sprintf("window.snipIdle&&window.snipIdle(%t)", monacoDark()))
}

func (p *panel) saveActive() {
	if p.chrome != nil && p.curPath != "" && !isPreview(p.curPath) {
		p.chrome.Eval(`window.snipGet&&window.snipGet()`)
	}
}

func (p *panel) hasDirty() bool {
	for _, d := range p.dirty {
		if d {
			return true
		}
	}
	return false
}

func (p *panel) canClose() bool {
	if !p.hasDirty() {
		return true
	}
	switch msgBox(p.hwnd, "Save unsaved notes?", "Editor", mbYesNoCancel) {
	case idYes:
		p.closing = true
		if p.chrome != nil {
			p.chrome.Eval(`window.snipSaveAll&&window.snipSaveAll()`)
			return false
		}
		return true
	case idNo:
		return true
	default:
		return false
	}
}

func (p *panel) openPath(rel string) {
	for i, t := range p.open {
		if t == rel {
			send(p.tabs, tcmSetCur, uintptr(i), 0)
			p.activate(i)
			return
		}
	}
	raw, err := snippet.Read(p.dataDir, rel)
	if err != nil {
		return
	}
	p.open = append(p.open, rel)
	p.insertTab(len(p.open)-1, tabTitle(rel, false))
	send(p.tabs, tcmSetCur, uintptr(len(p.open)-1), 0)
	p.curPath = rel
	p.pending = string(raw)
	setText(p.hwnd, "Editor — "+rel)
	p.push()
}

func (p *panel) activate(i int) {
	if i < 0 || i >= len(p.open) {
		return
	}
	p.curPath = p.open[i]
	if isPreview(p.curPath) {
		setText(p.hwnd, "Editor — Preview · "+previewSource(p.curPath))
		p.refreshPreview()
		return
	}
	setText(p.hwnd, "Editor — "+p.curPath)
	if p.chrome == nil || !p.ready {
		return
	}
	path, _ := json.Marshal(p.curPath)
	dark := monacoDark()
	p.chrome.Eval(fmt.Sprintf("window.snipSwitch&&window.snipSwitch(%s,%t)", string(path), dark))
}

func (p *panel) closePath(rel string) {
	if p.dirty[rel] {
		switch msgBox(p.hwnd, "Save "+rel+"?", "Editor", mbYesNoCancel) {
		case idYes:
			p.dropAfter = rel
			if p.curPath != rel {
				for i, t := range p.open {
					if t == rel {
						send(p.tabs, tcmSetCur, uintptr(i), 0)
						p.activate(i)
						break
					}
				}
			}
			p.saveActive()
			return
		case idNo:
			p.dirty[rel] = false
		default:
			return
		}
	}
	p.removeTab(rel)
}

func (p *panel) stripTab(rel string) {
	idx := -1
	for i, t := range p.open {
		if t == rel {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	send(p.tabs, tcmDelete, uintptr(idx), 0)
	p.open = append(p.open[:idx], p.open[idx+1:]...)
	delete(p.dirty, rel)
}

func (p *panel) removeTab(rel string) {
	if !isPreview(rel) {
		p.stripTab(previewOf(rel))
	}
	idx := -1
	for i, t := range p.open {
		if t == rel {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if p.chrome != nil && p.ready && !isPreview(rel) {
		path, _ := json.Marshal(rel)
		p.chrome.Eval(fmt.Sprintf("window.snipClose&&window.snipClose(%s)", string(path)))
	}
	send(p.tabs, tcmDelete, uintptr(idx), 0)
	p.open = append(p.open[:idx], p.open[idx+1:]...)
	delete(p.dirty, rel)
	if len(p.open) == 0 {
		p.curPath = ""
		setText(p.hwnd, "Editor")
		p.idlePane()
		return
	}
	next := idx
	if next >= len(p.open) {
		next = len(p.open) - 1
	}
	send(p.tabs, tcmSetCur, uintptr(next), 0)
	p.activate(next)
}

func (p *panel) insertTab(i int, title string) {
	u, _ := windows.UTF16FromString(title)
	item := tcitem{mask: tcifText, pszText: uintptr(unsafe.Pointer(&u[0]))}
	send(p.tabs, tcmInsert, uintptr(i), uintptr(unsafe.Pointer(&item)))
}

func (p *panel) refreshTab(rel string) {
	for i, t := range p.open {
		if t != rel {
			continue
		}
		u, _ := windows.UTF16FromString(tabTitle(rel, p.dirty[rel]))
		item := tcitem{mask: tcifText, pszText: uintptr(unsafe.Pointer(&u[0]))}
		send(p.tabs, tcmSetItem, uintptr(i), uintptr(unsafe.Pointer(&item)))
		return
	}
}

func tabTitle(rel string, dirty bool) string {
	if isPreview(rel) {
		return "Preview · " + path.Base(filepath.ToSlash(previewSource(rel)))
	}
	name := path.Base(filepath.ToSlash(rel))
	if dirty {
		return "*" + name
	}
	return name
}

func createParent(selPath string, isFile bool) string {
	if selPath == "" {
		return ""
	}
	if !isFile {
		return selPath
	}
	d := path.Dir(filepath.ToSlash(selPath))
	if d == "." {
		return ""
	}
	return d
}

func (p *panel) folderPrefix() string {
	return createParent(p.selPath, p.selFile)
}

func leafName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, `/\`) || strings.Contains(s, "..") {
		return "", false
	}
	return s, true
}

func joinRel(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func (p *panel) newFolder() {
	name, ok := askName(p.hwnd, "New folder", "")
	if !ok {
		return
	}
	name, ok = leafName(name)
	if !ok {
		return
	}
	rel := joinRel(p.folderPrefix(), name)
	if err := snippet.CreateDir(p.dataDir, rel); err != nil {
		return
	}
	p.selPath = rel
	p.selFile = false
	p.fillTree()
}

func (p *panel) newNote() {
	name, ok := askName(p.hwnd, "New note", "")
	if !ok {
		return
	}
	name, ok = leafName(name)
	if !ok {
		return
	}
	rel := joinRel(p.folderPrefix(), name)
	if err := snippet.CreateFile(p.dataDir, rel); err != nil {
		return
	}
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".md"
	}
	p.selPath = rel
	p.selFile = true
	p.fillTree()
	p.openPath(rel)
}

func (p *panel) deleteSel() {
	rel := p.selPath
	if rel == "" {
		return
	}
	msg := "Delete note " + rel + "?"
	if !p.selFile {
		msg = "Delete folder " + rel + " and everything in it?"
	}
	if msgBox(p.hwnd, msg, "Editor", mbYesNo) != idYes {
		return
	}
	if err := snippet.Remove(p.dataDir, rel); err != nil {
		return
	}
	open := append([]string{}, p.open...)
	for _, t := range open {
		src := t
		if isPreview(t) {
			src = previewSource(t)
		}
		if src == rel || strings.HasPrefix(src, rel+"/") {
			p.dirty[t] = false
			p.removeTab(t)
		}
	}
	p.selPath = ""
	p.selFile = false
	p.fillTree()
}

func remapRel(old, neu, p string) string {
	if p == old {
		return neu
	}
	if old != "" && strings.HasPrefix(p, old+"/") {
		return neu + p[len(old):]
	}
	return p
}

func remapStored(old, neu, stored string) string {
	if isPreview(stored) {
		return previewOf(remapRel(old, neu, previewSource(stored)))
	}
	return remapRel(old, neu, stored)
}

func (p *panel) renameSel() {
	rel := p.selPath
	if rel == "" {
		return
	}
	cur := path.Base(filepath.ToSlash(rel))
	name, ok := askName(p.hwnd, "Rename", cur)
	if !ok {
		return
	}
	name, ok = leafName(name)
	if !ok {
		return
	}
	if p.selFile && !strings.EqualFold(filepath.Ext(name), ".md") {
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ".md"
	}
	dest := joinRel(createParent(rel, p.selFile), name)
	if dest == rel {
		return
	}
	if err := snippet.Move(p.dataDir, rel, dest); err != nil {
		msg := "Can't rename there."
		if errors.Is(err, os.ErrExist) {
			msg = "A file or folder with that name already exists."
		}
		msgBox(p.hwnd, msg, "Editor", 0x10)
		return
	}
	p.afterMove(rel, dest)
}

func (p *panel) afterMove(old, neu string) {
	if p.chrome != nil && p.ready {
		for _, t := range p.open {
			if isPreview(t) {
				continue
			}
			n := remapRel(old, neu, t)
			if n == t {
				continue
			}
			oldJ, _ := json.Marshal(t)
			newJ, _ := json.Marshal(n)
			p.chrome.Eval(fmt.Sprintf("window.snipMove&&window.snipMove(%s,%s)", string(oldJ), string(newJ)))
		}
	}
	next := map[string]bool{}
	for k, v := range p.dirty {
		next[remapStored(old, neu, k)] = v
	}
	p.dirty = next
	for i, t := range p.open {
		p.open[i] = remapStored(old, neu, t)
	}
	p.selPath = remapRel(old, neu, p.selPath)
	p.curPath = remapStored(old, neu, p.curPath)
	p.fillTree()
	p.selectTree(p.selPath)
	for _, t := range p.open {
		p.refreshTab(t)
	}
	if p.curPath == "" {
		setText(p.hwnd, "Editor")
		return
	}
	if isPreview(p.curPath) {
		setText(p.hwnd, "Editor — Preview · "+previewSource(p.curPath))
		return
	}
	setText(p.hwnd, "Editor — "+p.curPath)
}

func askName(owner uintptr, title, initial string) (string, bool) {
	p := &panel{kind: "prompt", clicks: map[uintptr]func(){}, on: map[uintptr]bool{}}
	p.hwnd = create(owner, "", title, 0x90C80000, 200, 200, 360, 140, 0)
	applyFrame(p.hwnd)
	panels[p.hwnd] = p
	p.static(p.hwnd, "Name", 12, 12, 80, 18)
	p.dataEdit = p.edit(p.hwnd, "", 12, 36, 320, 24, false)
	if initial != "" {
		setText(p.dataEdit, initial)
	}
	ok := p.button(p.hwnd, "OK", 168, 72, 72, 26)
	cancel := p.button(p.hwnd, "Cancel", 248, 72, 80, 26)
	p.clicks[ok] = func() {
		p.pending = textOf(p.dataEdit)
		p.promptOK = true
		p.promptDone = true
		destroy(p.hwnd)
	}
	p.clicks[cancel] = func() {
		p.promptDone = true
		destroy(p.hwnd)
	}
	enableWin(owner, false)
	var msg wmsg
	for !p.promptDone {
		r, _, _ := pGetMsg.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 || int32(r) == -1 {
			break
		}
		pTrans.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
	enableWin(owner, true)
	focus(owner)
	return p.pending, p.promptOK
}

func (p *panel) onSize() {
	w, h := clientSize(p.hwnd)
	switch p.kind {
	case "help":
		if p.host != 0 {
			move(p.host, 8, 8, w-16, h-16)
		}
		if p.chrome != nil {
			p.chrome.Resize()
		}
		if p.dataEdit != 0 {
			move(p.dataEdit, 8, 8, w-16, h-16)
		}
	case "prefs":
		if p.dataEdit != 0 {
			move(p.dataEdit, 12, 32, w-120, 24)
		}
		if p.browse != 0 {
			move(p.browse, w-96, 30, 80, 26)
		}
		if p.bulk != 0 {
			move(p.bulk, 0, 170, w, h-170)
			p.layoutBulk()
		}
	case "editor":
		p.layoutBar()
		if p.tree != 0 {
			move(p.tree, 8, barH, sideW, h-barH-8)
		}
		if p.tabs != 0 {
			move(p.tabs, paneX, barH, w-paneX-8, tabH)
		}
		if p.host != 0 {
			move(p.host, paneX, barH+tabH, w-paneX-8, h-barH-tabH-8)
			if p.chrome != nil {
				p.chrome.Resize()
			}
		}
	}
}

func (p *panel) layoutBar() {
	w, _ := clientSize(p.hwnd)
	if p.btnDel != 0 {
		move(p.btnDel, 8+sideW-iconBtnS, 4, iconBtnS, iconBtnS)
	}
	if p.btnPreview != 0 {
		move(p.btnPreview, w-8-iconBtnS, 4, iconBtnS, iconBtnS)
	}
}

func frame(title string, w, h int32) uintptr {
	x, y := int32(80), int32(80)
	sw := metrics(0)
	sh := metrics(1)
	if sw > w {
		x = (sw - w) / 2
	}
	if sh > h {
		y = (sh - h) / 2
	}
	hwnd := create(0, "", title, wsOver|wsVis|wsClip, x, y, w, h, 0)
	if appIcon != 0 {
		send(hwnd, wmSetIcon, iconBig, appIcon)
	}
	if appIconSm != 0 {
		send(hwnd, wmSetIcon, iconSmall, appIconSm)
	}
	applyFrame(hwnd)
	return hwnd
}

func create(parent uintptr, class, title string, style uint32, x, y, w, h int32, id uintptr) uintptr {
	return createEx(0, parent, class, title, style, x, y, w, h, id)
}

func createEx(ex uint32, parent uintptr, class, title string, style uint32, x, y, w, h int32, id uintptr) uintptr {
	cls := class
	if cls == "" {
		cls = "snippertingFrame"
	}
	c, _ := windows.UTF16PtrFromString(cls)
	t, _ := windows.UTF16PtrFromString(title)
	par := parent
	if parent == 0 && class == "" && style == 0 {
		par = ^uintptr(2) // HWND_MESSAGE
	}
	hwnd, _, _ := pCreate.Call(uintptr(ex), uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), par, id, moduleHandle(), 0)
	if guiFont != 0 && hwnd != 0 {
		send(hwnd, wmSetFont, guiFont, 1)
	}
	return hwnd
}

func (p *panel) button(parent uintptr, title string, x, y, w, h int32) uintptr {
	return uintptr(create(parent, "BUTTON", title, wsChild|wsVis|wsTab|bsOwner, x, y, w, h, 0))
}

func (p *panel) pushLike(parent uintptr, title string, on bool) uintptr {
	h := uintptr(create(parent, "BUTTON", title, wsChild|wsVis|wsTab|bsOwner, 0, 0, 80, 24, 0))
	if p.on == nil {
		p.on = map[uintptr]bool{}
	}
	p.on[h] = on
	return h
}

func (p *panel) static(parent uintptr, title string, x, y, w, h int32) uintptr {
	return uintptr(create(parent, "STATIC", title, wsChild|wsVis, x, y, w, h, 0))
}

func (p *panel) edit(parent uintptr, title string, x, y, w, h int32, multi bool) uintptr {
	st := uint32(wsChild | wsVis | esAutoH | wsTab)
	if multi {
		st = wsChild | wsVis | wsVScroll | esMulti | esRead | esAutoV
	}
	hwnd := uintptr(createEx(0, parent, "EDIT", title, st, x, y, w, h, 0))
	colorEdit(hwnd)
	return hwnd
}

func textOf(h uintptr) string {
	n := send(h, wmGetTextLen, 0, 0)
	buf := make([]uint16, n+1)
	send(h, wmGetText, n+1, uintptr(unsafe.Pointer(&buf[0])))
	return windows.UTF16ToString(buf)
}

func setText(h uintptr, s string) {
	u, _ := windows.UTF16PtrFromString(s)
	send(h, 0x000C, 0, uintptr(unsafe.Pointer(u)))
}

func setScroll(hwnd uintptr, max int32, page uint32, pos int32) {
	raw := struct {
		cbSize, fMask   uint32
		nMin, nMax      int32
		nPage           uint32
		nPos, nTrackPos int32
	}{cbSize: 28, fMask: 7, nMax: max, nPage: page, nPos: pos}
	pScroll.Call(hwnd, 1, uintptr(unsafe.Pointer(&raw)), 1)
}

func findIco() string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands, filepath.Join(dir, "snipperting.ico"), filepath.Join(dir, "build", "snipperting.ico"))
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, filepath.Join(wd, "build", "snipperting.ico"))
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
