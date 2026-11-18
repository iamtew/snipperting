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
		t.Fatal("editor.html still uses the old HTTP prefix")
	}
	if !bytes.Contains(b, []byte("snipOpen")) {
		t.Fatal("editor.html must expose snipOpen")
	}
	if !bytes.Contains(b, []byte("snipSaveAll")) {
		t.Fatal("editor.html must expose snipSaveAll")
	}
	if !bytes.Contains(b, []byte("snipFmt")) {
		t.Fatal("editor.html must expose snipFmt")
	}
}
