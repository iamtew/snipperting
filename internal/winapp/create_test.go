package winapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNoteMD(t *testing.T) {
	if noteMD("hello") != "hello.md" {
		t.Fatal("omit")
	}
	if noteMD("hello.md") != "hello.md" {
		t.Fatal("keep")
	}
	if noteMD("keep.bak") != "keep.bak.md" {
		t.Fatal("append")
	}
}

func TestUniqueLeaf(t *testing.T) {
	taken := map[string]bool{"untitled.md": true, "new folder": true}
	if uniqueLeaf(taken, "note.md") != "note.md" {
		t.Fatal("free")
	}
	if uniqueLeaf(taken, "untitled.md") != "untitled-2.md" {
		t.Fatal("file")
	}
	if uniqueLeaf(taken, "New folder") != "New folder-2" {
		t.Fatal("dir")
	}
}

func TestNoteStampName(t *testing.T) {
	tm := time.Date(2026, 11, 18, 21, 39, 0, 0, time.Local)
	if got := noteStampName(tm); got != "note-1118-2139.md" {
		t.Fatal(got)
	}
}

func TestUniqueRootName(t *testing.T) {
	dir := t.TempDir()
	if uniqueRootName(dir, "note-1118-2139.md") != "note-1118-2139.md" {
		t.Fatal("free")
	}
	if err := os.WriteFile(filepath.Join(dir, "note-1118-2139.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := uniqueRootName(dir, "note-1118-2139.md"); got != "note-1118-2139-2.md" {
		t.Fatal(got)
	}
}

func TestItemDir(t *testing.T) {
	if itemDir("Folder123") != "" {
		t.Fatal("root folder")
	}
	if itemDir("Folder123/sub") != "Folder123" {
		t.Fatal("nested folder")
	}
	if itemDir("Folder123/a.md") != "Folder123" {
		t.Fatal("file")
	}
	if itemDir("a.md") != "" {
		t.Fatal("root file")
	}
}

func TestCreateParent(t *testing.T) {
	cases := []struct {
		path string
		file bool
		want string
	}{
		{"", false, ""},
		{"", true, ""},
		{"Folder1", false, "Folder1"},
		{"note.md", true, ""},
		{"Folder1/lala123.md", true, "Folder1"},
		{"a/b/c.md", true, "a/b"},
	}
	for _, c := range cases {
		if got := createParent(c.path, c.file); got != c.want {
			t.Fatalf("createParent(%q, %v)=%q want %q", c.path, c.file, got, c.want)
		}
	}
}

func TestMoveDest(t *testing.T) {
	if moveDest("a.md", "sub", false, true) != "sub/a.md" {
		t.Fatal("folder")
	}
	if moveDest("a.md", "sub/b.md", true, true) != "sub/a.md" {
		t.Fatal("file parent")
	}
	if moveDest("sub/a.md", "", false, false) != "a.md" {
		t.Fatal("root")
	}
	if moveDest("x/y.md", "x/y.md", true, true) != "x/y.md" {
		t.Fatal("self")
	}
}

func TestRemapRel(t *testing.T) {
	if remapRel("a.md", "b.md", "a.md") != "b.md" {
		t.Fatal("file")
	}
	if remapRel("old", "neu", "old/n.md") != "neu/n.md" {
		t.Fatal("child")
	}
	if remapStored("a.md", "b.md", previewOf("a.md")) != previewOf("b.md") {
		t.Fatal("preview")
	}
	if remapRel("a", "b", "other.md") != "other.md" {
		t.Fatal("other")
	}
}

func TestPreviewTab(t *testing.T) {
	if isPreview("note.md") || !isPreview(previewOf("note.md")) {
		t.Fatal("isPreview")
	}
	if previewSource(previewOf("a/b.md")) != "a/b.md" {
		t.Fatal("previewSource")
	}
	if tabTitle(previewOf("folder/note.md"), true) != "Preview · note.md" {
		t.Fatal(tabTitle(previewOf("folder/note.md"), true))
	}
}
