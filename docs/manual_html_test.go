package docs

import (
	"bytes"
	"os"
	"testing"
)

func TestManualHTMLLoadsMarkdown(t *testing.T) {
	b, err := os.ReadFile("manual.html")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("MANUAL.md")) {
		t.Fatal("manual.html must fetch MANUAL.md")
	}
	if !bytes.Contains(b, []byte("vendor/marked.min.js")) {
		t.Fatal("manual.html must load vendored marked")
	}
}
