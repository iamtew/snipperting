package snippet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHeader(t *testing.T) {
	raw := []byte(`----
label: Demo Snippet
about: alt-text
action: inject
state: disable
hidden: true
----
# Title
body line
`)
	d := Parse(raw)
	if d.Meta.Label != "Demo Snippet" || d.Meta.About != "alt-text" {
		t.Fatalf("meta %+v", d.Meta)
	}
	if d.Meta.Action != "inject" || d.Meta.Enabled || !d.Meta.Hidden {
		t.Fatalf("flags %+v", d.Meta)
	}
	if !strings.HasPrefix(d.Body, "# Title") || !strings.Contains(d.Body, "body line") {
		t.Fatalf("body %q", d.Body)
	}
}

func TestParseNoHeader(t *testing.T) {
	d := Parse([]byte("just text\n"))
	if d.Body != "just text\n" {
		t.Fatalf("body %q", d.Body)
	}
	if !d.Meta.Enabled || d.Meta.Action != "clipboard" {
		t.Fatalf("defaults %+v", d.Meta)
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	m := Meta{Label: "A", About: "B", Action: "inject", Enabled: false, Hidden: true}
	got := Parse(Encode(m, "hello"))
	if got.Meta != m || got.Body != "hello\n" {
		t.Fatalf("got %+v body %q", got.Meta, got.Body)
	}
}

func TestWalkAndJoin(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), Encode(Meta{Label: "Alpha", Action: "clipboard", Enabled: true}, "aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work", "b.md"), Encode(Meta{Label: "Beta", Action: "inject", Enabled: true, Hidden: true}, "bbb"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Children) != 3 {
		t.Fatalf("children %+v", tree.Children)
	}
	if _, err := JoinUnder(dir, "../secret.md"); err == nil {
		t.Fatal("expected reject")
	}
	p, err := JoinUnder(dir, "work/b.md")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "b.md" {
		t.Fatalf("path %s", p)
	}
}

func TestCreateDirAndFile(t *testing.T) {
	dir := t.TempDir()
	if err := CreateDir(dir, "notes"); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(dir, "notes/hello"); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(dir, "notes/hello.md"); err != os.ErrExist {
		t.Fatalf("exist %v", err)
	}
	if err := CreateFile(dir, "../nope.md"); err == nil {
		t.Fatal("expected reject")
	}
	raw, err := Read(dir, "notes/hello.md")
	if err != nil {
		t.Fatal(err)
	}
	d := Parse(raw)
	if d.Meta.Action != "clipboard" || !d.Meta.Enabled {
		t.Fatalf("header %+v", d.Meta)
	}
}

func TestStateSynonyms(t *testing.T) {
	d := Parse([]byte("----\nstate: off\nhidden: yes\n----\nx\n"))
	if d.Meta.Enabled || !d.Meta.Hidden {
		t.Fatalf("%+v", d.Meta)
	}
}
