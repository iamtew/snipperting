package winapp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"snipperting/internal/config"
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
	wmSetFont    = 0x0030
	wmGetText    = 0x000D
	wmGetTextLen = 0x000E

	wsOver    = 0x00CF0000
	wsVis     = 0x10000000
	wsChild   = 0x40000000
	wsVScroll = 0x00200000
	wsBorder  = 0x00800000
	wsTab     = 0x00010000
	wsClip    = 0x02000000

	esMulti = 0x0004
	esRead  = 0x0800
	esAutoV = 0x0040
	esAutoH = 0x0080

	tviLast       = ^uintptr(0xFFFD) // (HTREEITEM)-0xFFFE
	tviRoot       = ^uintptr(0xFFFF) // (HTREEITEM)-0x10000
	tvmInsert     = 0x1132
	tvmDelete     = 0x1101
	tvifTextParam = 0x0005
	tvnSelChanged = ^uint32(450) // TVN_SELCHANGEDW = -451

	rowH = 28
)

var (
	uiOnce   sync.Once
	uiMu     sync.Mutex
	uiHwnd   uintptr
	uiQueue  []openReq
	uiCfg    *config.Store
	uiPort   int
	uiManual string
	uiVer    string
	panels   = map[uintptr]*panel{}
	classU   []uint16
	wndCB    uintptr
	guiFont  uintptr
)

type openReq struct{ kind, file string }

type tnode struct {
	path string
	file bool
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

type panel struct {
	kind     string
	hwnd     uintptr
	clicks   map[uintptr]func()
	dataEdit uintptr
	bulk     uintptr
	placed   []placed
	rows     int
	scroll   int32
	opened   map[string]bool
	tree     uintptr
	host     uintptr
	nodes    []tnode
	curPath  string
	pending  string
	ready    bool
	chrome   *edge.Chromium
	dataDir  string
}

func StartUI(cfg *config.Store, port int, manual, version string) {
	uiOnce.Do(func() {
		uiCfg = cfg
		uiPort = port
		uiManual = manual
		uiVer = version
		ready := make(chan struct{})
		go func() {
			runtime.LockOSThread()
			initControls()
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
	var icc struct{ size, flags uint32 }
	icc.size = 8
	icc.flags = 0xFF
	windows.NewLazySystemDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&icc)))
	h, _, _ := windows.NewLazySystemDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	guiFont = h
}

func registerFrame() {
	classU, _ = windows.UTF16FromString("snippertingFrame")
	wndCB = windows.NewCallback(wndProc)
	wc := wclass{
		size:      uint32(unsafe.Sizeof(wclass{})),
		wndProc:   wndCB,
		instance:  moduleHandle(),
		bg:        6,
		className: &classU[0],
	}
	pReg.Call(uintptr(unsafe.Pointer(&wc)))
}

func createPump() {
	h := create(0, "", "", 0, 0, 0, 0, 0, 0)
	uiMu.Lock()
	uiHwnd = uintptr(h)
	uiMu.Unlock()
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
	if msg == wmOpen {
		drain()
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
			p.onTree(lp)
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
			destroy(hwnd)
			return true
		}
	case wmDestroy:
		if hwnd == p.hwnd {
			delete(panels, hwnd)
			if p.bulk != 0 {
				delete(panels, p.bulk)
			}
			p.hwnd = 0
		}
	}
	return false
}

func openPrefs() {
	p := &panel{kind: "prefs", clicks: map[uintptr]func(){}, opened: map[string]bool{}}
	p.hwnd = uintptr(frame("Preferences", 720, 560))
	panels[p.hwnd] = p
	p.static(p.hwnd, "Data directory", 12, 12, 200, 18)
	p.dataEdit = p.edit(p.hwnd, "", 12, 32, 580, 24, false)
	save := p.button(p.hwnd, "Save", 600, 30, 80, 26)
	p.clicks[save] = p.savePrefs
	p.static(p.hwnd, "Bulk Edit", 12, 68, 200, 18)
	p.bulk = uintptr(create(p.hwnd, "", "", wsChild|wsVis|wsVScroll|wsClip, 0, 90, 704, 430, 0))
	panels[p.bulk] = p
	p.rebuildPrefs()
}

func (p *panel) savePrefs() {
	cur := uiCfg.Get()
	_ = uiCfg.Set(config.File{DataDir: textOf(p.dataEdit), Theme: cur.Theme})
	p.rebuildPrefs()
}

func (p *panel) rebuildPrefs() {
	for _, pl := range p.placed {
		destroy(pl.hwnd)
		delete(p.clicks, pl.hwnd)
	}
	p.placed = nil
	p.rows = 0
	p.scroll = 0
	cur := uiCfg.Get()
	setText(p.dataEdit, cur.DataDir)
	tree, err := snippet.Walk(cur.DataDir)
	if err == nil && len(tree.Children) > 0 {
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
		label := n.Name
		if n.Label != "" {
			label += " — " + n.Label
		}
		lab := p.static(p.bulk, label, 0, 0, 200, 24)
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
		ab := p.button(p.bulk, act, 0, 0, 88, 24)
		sb := p.button(p.bulk, st, 0, 0, 64, 24)
		hb := p.button(p.bulk, hid, 0, 0, 72, 24)
		p.placed = append(p.placed,
			placed{hwnd: ab, row: row, col: 1},
			placed{hwnd: sb, row: row, col: 2},
			placed{hwnd: hb, row: row, col: 3},
		)
		p.clicks[ab] = func() {
			next := "inject"
			if act == "inject" {
				next = "clipboard"
			}
			_ = snippet.UpdateMeta(uiCfg.Get().DataDir, path, func(m *snippet.Meta) { m.Action = next })
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
			move(pl.hwnd, pl.x, y, w-pl.x-240, rowH-4)
		case 1:
			move(pl.hwnd, w-236, y, 88, rowH-4)
		case 2:
			move(pl.hwnd, w-144, y, 64, rowH-4)
		case 3:
			move(pl.hwnd, w-76, y, 72, rowH-4)
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
	ed := p.edit(p.hwnd, "", 8, 8, 688, 584, true)
	setText(ed, uiManual)
	send(ed, 0x00C5, 0x7ffffffe, 0)
	p.dataEdit = ed
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
	p.static(p.hwnd, "Snippets from the Windows tray.", 24, 110, 360, 40)
}

func openEditor(file string) {
	p := &panel{kind: "editor", clicks: map[uintptr]func(){}, dataDir: uiCfg.Get().DataDir}
	p.hwnd = uintptr(frame("Editor", 1100, 720))
	panels[p.hwnd] = p
	save := p.button(p.hwnd, "Save", 8, 6, 72, 26)
	p.clicks[save] = func() {
		if p.chrome != nil && p.curPath != "" {
			p.chrome.Eval(`window.snipGet&&window.snipGet()`)
		}
	}
	p.tree = uintptr(create(p.hwnd, "SysTreeView32", "", wsChild|wsVis|wsBorder|0x0027, 8, 40, 260, 640, 0))
	p.host = uintptr(create(p.hwnd, "", "", wsChild|wsVis|wsClip, 276, 40, 800, 640, 0))
	p.fillTree()
	p.chrome = edge.NewChromium()
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		p.chrome.DataPath = filepath.Join(local, "snipperting", "webview")
	}
	p.chrome.MessageCallback = func(msg string) {
		var m struct {
			T string `json:"t"`
			V string `json:"v"`
			P string `json:"p"`
		}
		if json.Unmarshal([]byte(msg), &m) != nil {
			return
		}
		if m.T == "ready" {
			p.ready = true
			p.push()
			return
		}
		if m.T == "save" {
			rel := m.P
			if rel == "" {
				rel = p.curPath
			}
			if rel != "" {
				_ = snippet.Write(p.dataDir, rel, []byte(m.V))
			}
		}
	}
	if !p.chrome.Embed(p.host) {
		p.static(p.host, "WebView2 failed to start.", 8, 8, 400, 24)
		p.chrome = nil
		return
	}
	p.chrome.Resize()
	p.chrome.Navigate(fmt.Sprintf("http://127.0.0.1:%d/editor", uiPort))
	if file != "" {
		p.loadFile(file)
	}
}

func (p *panel) fillTree() {
	if p.tree == 0 {
		return
	}
	send(p.tree, tvmDelete, 0, tviRoot)
	p.nodes = nil
	tree, err := snippet.Walk(p.dataDir)
	if err != nil {
		return
	}
	p.addTree(tviRoot, tree.Children)
}

func (p *panel) addTree(parent uintptr, nodes []snippet.Node) {
	for _, n := range nodes {
		if n.Type == "file" && n.Hidden {
			continue
		}
		label := n.Label
		if n.Type == "dir" {
			label = n.Name
		}
		p.nodes = append(p.nodes, tnode{path: n.Path, file: n.Type == "file"})
		h := p.insertTree(parent, label, uintptr(len(p.nodes)-1))
		if n.Type == "dir" {
			p.addTree(h, n.Children)
		}
	}
}

func (p *panel) insertTree(parent uintptr, label string, idx uintptr) uintptr {
	u, _ := windows.UTF16FromString(label)
	ins := tvinsert{
		hParent:      parent,
		hInsertAfter: tviLast,
		item: tvitem{
			mask:    tvifTextParam,
			pszText: uintptr(unsafe.Pointer(&u[0])),
			lParam:  idx,
		},
	}
	return send(p.tree, tvmInsert, 0, uintptr(unsafe.Pointer(&ins)))
}

func (p *panel) onTree(lp uintptr) {
	nm := (*nmtv)(unsafe.Pointer(lp))
	if nm.hdr.code != tvnSelChanged {
		return
	}
	i := int(nm.itemNew.lParam)
	if i < 0 || i >= len(p.nodes) || !p.nodes[i].file {
		return
	}
	p.loadFile(p.nodes[i].path)
}

func (p *panel) loadFile(rel string) {
	raw, err := snippet.Read(p.dataDir, rel)
	if err != nil {
		return
	}
	p.curPath = rel
	p.pending = string(raw)
	setText(p.hwnd, "Editor — "+rel)
	p.push()
}

func (p *panel) push() {
	if p.chrome == nil || !p.ready {
		return
	}
	b, _ := json.Marshal(p.pending)
	dark := monacoDark("system")
	path, _ := json.Marshal(p.curPath)
	if p.curPath != "" {
		p.chrome.Eval(fmt.Sprintf("window.snipOpen&&window.snipOpen(%s,%s,%t)", string(path), string(b), dark))
		return
	}
	p.chrome.Eval(fmt.Sprintf("window.snipSet&&window.snipSet(%s,%t)", string(b), dark))
}

func (p *panel) onSize() {
	w, h := clientSize(p.hwnd)
	switch p.kind {
	case "help":
		if p.dataEdit != 0 {
			move(p.dataEdit, 8, 8, w-16, h-16)
		}
	case "prefs":
		if p.dataEdit != 0 {
			move(p.dataEdit, 12, 32, w-120, 24)
		}
		if p.bulk != 0 {
			move(p.bulk, 0, 90, w, h-90)
			p.layoutBulk()
		}
	case "editor":
		if p.tree != 0 {
			move(p.tree, 8, 40, 260, h-48)
		}
		if p.host != 0 {
			move(p.host, 276, 40, w-284, h-48)
			if p.chrome != nil {
				p.chrome.Resize()
			}
		}
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
	return create(0, "", title, wsOver|wsVis|wsClip, x, y, w, h, 0)
}

func create(parent uintptr, class, title string, style uint32, x, y, w, h int32, id uintptr) uintptr {
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
	hwnd, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), par, id, moduleHandle(), 0)
	if guiFont != 0 && hwnd != 0 {
		send(hwnd, wmSetFont, guiFont, 1)
	}
	return hwnd
}

func (p *panel) button(parent uintptr, title string, x, y, w, h int32) uintptr {
	return uintptr(create(parent, "BUTTON", title, wsChild|wsVis|wsTab, x, y, w, h, 0))
}

func (p *panel) static(parent uintptr, title string, x, y, w, h int32) uintptr {
	return uintptr(create(parent, "STATIC", title, wsChild|wsVis, x, y, w, h, 0))
}

func (p *panel) edit(parent uintptr, title string, x, y, w, h int32, multi bool) uintptr {
	st := uint32(wsChild | wsVis | wsBorder | esAutoH | wsTab)
	if multi {
		st = wsChild | wsVis | wsBorder | wsVScroll | esMulti | esRead | esAutoV
	}
	return uintptr(create(parent, "EDIT", title, st, x, y, w, h, 0))
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

func monacoDark(theme string) bool {
	switch theme {
	case "dark":
		return true
	case "light":
		return false
	default:
		k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
		if err != nil {
			return false
		}
		defer k.Close()
		v, _, err := k.GetIntegerValue("AppsUseLightTheme")
		return err == nil && v == 0
	}
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
