package winapp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEditorHTMLHasWorker(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "editor.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("workerMain.js")) {
		t.Fatal("editor.html must pin Monaco workers")
	}
	if !bytes.Contains(b, []byte("/vendor/monaco")) {
		t.Fatal("editor.html must load Monaco from the mapped web root")
	}
	if bytes.Contains(b, []byte("/web/vendor/monaco")) {
		t.Fatal("editor.html must not use /web/vendor/monaco paths")
	}
	if !bytes.Contains(b, []byte("snipOpen")) {
		t.Fatal("editor.html must expose snipOpen")
	}
	if !bytes.Contains(b, []byte("bodyEnd")) || !bytes.Contains(b, []byte("setPosition")) {
		t.Fatal("editor.html snipOpen must place caret at body end when asked")
	}
	if !bytes.Contains(b, []byte("snipSaveAll")) {
		t.Fatal("editor.html must expose snipSaveAll")
	}
	if !bytes.Contains(b, []byte("snipFmt")) {
		t.Fatal("editor.html must expose snipFmt")
	}
	if !bytes.Contains(b, []byte("snipPreview")) {
		t.Fatal("editor.html must expose snipPreview")
	}
	if !bytes.Contains(b, []byte("snipIdle")) {
		t.Fatal("editor.html must expose snipIdle")
	}
	if !bytes.Contains(b, []byte("snipMove")) {
		t.Fatal("editor.html must expose snipMove")
	}
	if !bytes.Contains(b, []byte("Open a note from the list on the left.")) {
		t.Fatal("editor.html must show empty-pane copy")
	}
	if !bytes.Contains(b, []byte(`id="s"`)) {
		t.Fatal("editor.html must have a status bar")
	}
	if !bytes.Contains(b, []byte("Ln ")) || !bytes.Contains(b, []byte("UTF-8")) || !bytes.Contains(b, []byte("Markdown")) {
		t.Fatal("editor.html status bar must show Ln, UTF-8, Markdown")
	}
}
