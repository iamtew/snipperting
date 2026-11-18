package winapp

import "testing"

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
