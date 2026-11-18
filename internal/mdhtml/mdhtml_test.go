package mdhtml

import (
	"strings"
	"testing"
)

func TestInlineShotURIs(t *testing.T) {
	md := "see ![Editor](shots/editor.png) and [site](https://example.com)"
	out := InlineShotURIs(md, map[string][]byte{"shots/editor.png": []byte("PNG")})
	if !strings.Contains(out, "data:image/png;base64,UE5H") {
		t.Fatal(out)
	}
	if strings.Contains(out, "](shots/editor.png)") {
		t.Fatal("editor path left")
	}
	if !strings.Contains(out, "https://example.com") {
		t.Fatal("other link")
	}
}

func TestRenderAndTOC(t *testing.T) {
	md := `# Title

intro **bold** text

## Install

steps

## Data folder

` + "```\ncode here\n```" + `

## Snippet files
`
	html := HTML(md)
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Fatalf("bold: %s", html)
	}
	if !strings.Contains(html, "<pre") || !strings.Contains(html, "code here") {
		t.Fatalf("fence: %s", html)
	}
	heads := headings(md)
	if len(heads) != 3 {
		t.Fatalf("toc %#v", heads)
	}
	page := Page("Help", md, false)
	for _, h := range heads {
		if h.ID == "" {
			t.Fatalf("empty id %#v", h)
		}
		if !strings.Contains(page, `href="#`+h.ID+`"`) {
			t.Fatalf("toc link %s missing in %s", h.ID, page)
		}
		if !strings.Contains(page, `id="`+h.ID+`"`) {
			t.Fatalf("heading id %s missing in %s", h.ID, page)
		}
	}
	if strings.Contains(page, `href="#title"`) {
		t.Fatal("h1 should not be in toc")
	}
	h1 := strings.Index(page, "<h1")
	nav := strings.Index(page, `<nav class="toc"`)
	h2 := strings.Index(page, "<h2")
	if h1 < 0 || nav < 0 || h2 < 0 || !(h1 < nav && nav < h2) {
		t.Fatalf("want h1 then toc then h2, got h1=%d toc=%d h2=%d", h1, nav, h2)
	}
}
