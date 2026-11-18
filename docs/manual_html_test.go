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

func TestPagesGitHubLinks(t *testing.T) {
	need := [][]byte{
		[]byte("https://github.com/iamtew/snipperting"),
		[]byte("https://github.com/iamtew/snipperting/releases/latest"),
		[]byte("https://github.com/iamtew/snipperting/issues"),
		[]byte("vendor/tabler/"),
	}
	for _, name := range []string{"index.html", "manual.html"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range need {
			if !bytes.Contains(b, n) {
				t.Fatalf("%s must contain %s", name, n)
			}
		}
	}
}
