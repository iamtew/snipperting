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
