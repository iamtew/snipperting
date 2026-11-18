package winapp

import "testing"

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
