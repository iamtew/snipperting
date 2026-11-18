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
