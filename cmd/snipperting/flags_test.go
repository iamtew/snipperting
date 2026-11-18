package main

import "testing"

func TestParseOpen(t *testing.T) {
	p, e, f := parseOpen(nil)
	if p || e || f != "" {
		t.Fatal("empty")
	}
	p, e, f = parseOpen([]string{"--prefs", "--edit"})
	if !p || !e || f != "" {
		t.Fatal("both, no file")
	}
	p, e, f = parseOpen([]string{"--edit", "a.md", "--prefs"})
	if !p || !e || f != "a.md" {
		t.Fatal(p, e, f)
	}
	p, e, f = parseOpen([]string{"--edit", "--prefs"})
	if !p || !e || f != "" {
		t.Fatal("edit then prefs is not a file")
	}
	p, e, f = parseOpen([]string{"--edit=rel.md"})
	if p || !e || f != "rel.md" {
		t.Fatal("equals form", p, e, f)
	}
}
