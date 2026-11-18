package winapp

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLabelEnterCreatesFile(t *testing.T) {
	initControls()
	registerFrame()
	dir := t.TempDir()
	hwnd := frame("label-test", 420, 360)
	p := &panel{
		kind:    "editor",
		hwnd:    hwnd,
		dataDir: dir,
		clicks:  map[uintptr]func(){},
		dirty:   map[string]bool{},
	}
	panels[hwnd] = p
	defer func() {
		delete(panels, hwnd)
		destroy(hwnd)
	}()
	tree := uintptr(create(hwnd, "SysTreeView32", "", wsChild|wsVis|tvsHasButtons|tvsEditLabels|tvsShowSelAlways|tvsFullRowSelect, 0, 0, 400, 320, 0))
	if tree == 0 {
		t.Fatal("tree")
	}
	p.tree = tree
	u, _ := windows.UTF16FromString("untitled.md")
	ins := tvinsert{
		hParent:      tviRoot,
		hInsertAfter: tviLast,
		item: tvitem{
			mask:    tvifIcon,
			pszText: uintptr(unsafe.Pointer(&u[0])),
			lParam:  0,
		},
	}
	item := send(tree, tvmInsert, 0, uintptr(unsafe.Pointer(&ins)))
	if item == 0 {
		t.Fatal("insert")
	}
	p.nodes = []tnode{{path: "untitled.md", file: true, item: item, creating: true}}
	p.startEdit(item)
	pump()
	edit := send(tree, tvmGetEdit, 0, 0)
	if edit == 0 {
		t.Fatal("no in-place edit control")
	}
	setText(edit, "kept.md")
	send(edit, wmKeyDown, vkReturn, 0)
	for i := 0; i < 30; i++ {
		pump()
	}
	if _, err := os.Stat(filepath.Join(dir, "kept.md")); err != nil {
		t.Fatalf("Enter did not create the note: typed=%q dest=%q err=%v", p.labelTyped, p.labelDest, err)
	}
}

func TestLabelEnterRenamesFile(t *testing.T) {
	initControls()
	registerFrame()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sib-a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sib-b.md"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hwnd := frame("label-rename", 420, 360)
	p := &panel{
		kind:    "editor",
		hwnd:    hwnd,
		dataDir: dir,
		clicks:  map[uintptr]func(){},
		dirty:   map[string]bool{},
	}
	panels[hwnd] = p
	defer func() {
		delete(panels, hwnd)
		destroy(hwnd)
	}()
	tree := uintptr(create(hwnd, "SysTreeView32", "", wsChild|wsVis|tvsHasButtons|tvsEditLabels|tvsShowSelAlways|tvsFullRowSelect, 0, 0, 400, 320, 0))
	p.tree = tree
	u, _ := windows.UTF16FromString("old.md")
	ins := tvinsert{
		hParent:      tviRoot,
		hInsertAfter: tviLast,
		item: tvitem{
			mask:    tvifIcon,
			pszText: uintptr(unsafe.Pointer(&u[0])),
			lParam:  0,
		},
	}
	item := send(tree, tvmInsert, 0, uintptr(unsafe.Pointer(&ins)))
	p.nodes = []tnode{{path: "old.md", file: true, item: item}}
	p.startEdit(item)
	pump()
	edit := send(tree, tvmGetEdit, 0, 0)
	if edit == 0 {
		t.Fatal("no in-place edit control")
	}
	setText(edit, "new.md")
	send(edit, wmKeyDown, vkReturn, 0)
	for i := 0; i < 30; i++ {
		pump()
	}
	if _, err := os.Stat(filepath.Join(dir, "new.md")); err != nil {
		t.Fatalf("Enter did not rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.md")); !os.IsNotExist(err) {
		t.Fatal("old name still on disk")
	}
	for _, tpath := range p.open {
		if tpath == "sib-a.md" || tpath == "sib-b.md" {
			t.Fatalf("rename opened sibling notes: %v", p.open)
		}
	}
}

func TestLabelEnterRenamesFolder(t *testing.T) {
	initControls()
	registerFrame()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Folder123"), 0o755); err != nil {
		t.Fatal(err)
	}
	hwnd := frame("label-folder", 420, 360)
	p := &panel{
		kind:    "editor",
		hwnd:    hwnd,
		dataDir: dir,
		clicks:  map[uintptr]func(){},
		dirty:   map[string]bool{},
	}
	panels[hwnd] = p
	defer func() {
		delete(panels, hwnd)
		destroy(hwnd)
	}()
	tree := uintptr(create(hwnd, "SysTreeView32", "", wsChild|wsVis|tvsHasButtons|tvsEditLabels|tvsShowSelAlways|tvsFullRowSelect, 0, 0, 400, 320, 0))
	p.tree = tree
	u, _ := windows.UTF16FromString("Folder123")
	ins := tvinsert{
		hParent:      tviRoot,
		hInsertAfter: tviLast,
		item: tvitem{
			mask:    tvifIcon,
			pszText: uintptr(unsafe.Pointer(&u[0])),
			lParam:  0,
		},
	}
	item := send(tree, tvmInsert, 0, uintptr(unsafe.Pointer(&ins)))
	p.nodes = []tnode{{path: "Folder123", file: false, item: item}}
	p.startEdit(item)
	pump()
	edit := send(tree, tvmGetEdit, 0, 0)
	if edit == 0 {
		t.Fatal("no in-place edit control")
	}
	setText(edit, "Renamed")
	send(edit, wmKeyDown, vkReturn, 0)
	send(edit, wmChar, vkReturn, 0)
	for i := 0; i < 30; i++ {
		pump()
	}
	if _, err := os.Stat(filepath.Join(dir, "Renamed")); err != nil {
		t.Fatalf("Enter did not rename folder: dest would have been %q err=%v", joinRel(itemDir("Folder123"), "Renamed"), err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Folder123")); !os.IsNotExist(err) {
		t.Fatal("old folder still on disk")
	}
	if _, err := os.Stat(filepath.Join(dir, "Folder123", "Renamed")); err == nil {
		t.Fatal("folder was moved into itself")
	}
}
