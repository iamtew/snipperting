package mdhtml

import (
	"strings"
	"testing"
)

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
	toc := TOC(md)
	if len(toc) != 3 {
		t.Fatalf("toc %#v", toc)
	}
	page := Page("Help", md, false)
	for _, h := range toc {
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
