package snippet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	if err := CreateFile(dir, "a.md"); err != nil {
		t.Fatal(err)
	}
	if err := CreateDir(dir, "sub"); err != nil {
		t.Fatal(err)
	}
	if err := Move(dir, "a.md", "sub/a.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("src still there")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "a.md")); err != nil {
		t.Fatal(err)
	}
}

func TestMoveRejects(t *testing.T) {
	dir := t.TempDir()
	if err := CreateDir(dir, "a"); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(dir, "a/n.md"); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(dir, "b.md"); err != nil {
		t.Fatal(err)
	}
	if err := Move(dir, "a", "a/x"); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("into self: %v", err)
	}
	if err := Move(dir, "b.md", "a/n.md"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("exist: %v", err)
	}
	if err := Move(dir, "b.md", "../out.md"); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("escape: %v", err)
	}
}
