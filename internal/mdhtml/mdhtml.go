package mdhtml

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var gm = goldmark.New(
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

type Heading struct {
	Title string
	ID    string
}

func HTML(md string) string {
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "<pre>" + html.EscapeString(md) + "</pre>"
	}
	return buf.String()
}

func TOC(md string) []Heading {
	src := []byte(md)
	doc := gm.Parser().Parse(text.NewReader(src))
	var out []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Level < 2 {
			return ast.WalkContinue, nil
		}
		id := attrID(h)
		title := strings.TrimSpace(string(h.Text(src)))
		out = append(out, Heading{Title: title, ID: id})
		return ast.WalkContinue, nil
	})
	return out
}

func attrID(n ast.Node) string {
	v, ok := n.AttributeString("id")
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func Page(title, md string, dark bool) string {
	bg, fg, muted, chip, accent := "#f3f3f3", "#1a1a1a", "#5c5c5c", "#ececec", "#0067c0"
	if dark {
		bg, fg, muted, chip, accent = "#202020", "#f3f3f3", "#b0b0b0", "#2b2b2b", "#60cdff"
	}
	var toc strings.Builder
	items := TOC(md)
	if len(items) > 0 {
		toc.WriteString(`<nav class="toc"><p>Contents</p><ul>`)
		for _, it := range items {
			fmt.Fprintf(&toc, `<li><a href="#%s">%s</a></li>`, html.EscapeString(it.ID), html.EscapeString(it.Title))
		}
		toc.WriteString(`</ul></nav>`)
	}
	preamble, rest := splitPreamble(HTML(md))
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>%s</title>
<style>
html,body{margin:0;background:%s;color:%s;font:16px/1.55 "Segoe UI",system-ui,sans-serif}
main{max-width:42rem;margin:0 auto;padding:1.5rem 1.75rem 3rem}
.toc{margin:1.25rem 0 2rem;padding:1rem 1.2rem;background:%s;border-radius:8px}
.toc p{margin:0 0 .5rem;font-weight:600}
.toc ul{margin:0;padding-left:1.2rem}
.toc a{color:%s;text-decoration:none}
.toc a:hover{text-decoration:underline}
h1{font-size:1.8rem;line-height:1.25}
h2{margin-top:1.8rem;border-bottom:1px solid %s;padding-bottom:.3rem}
a{color:%s}
code{font-family:Consolas,"Cascadia Code",monospace;background:%s;padding:.1em .35em;border-radius:4px;font-size:.9em}
pre{background:%s;padding:1rem;border-radius:8px;overflow:auto}
pre code{background:none;padding:0}
blockquote{margin:1rem 0;padding:.2rem 1rem;border-left:4px solid %s;color:%s}
</style></head><body><main><article>%s%s%s</article></main></body></html>`,
		html.EscapeString(title), bg, fg, chip, accent, muted, accent, chip, chip, accent, muted, preamble, toc.String(), rest)
}

// splitPreamble cuts rendered markdown at the first H2 so the title and intro sit above the TOC.
func splitPreamble(body string) (preamble, rest string) {
	i := strings.Index(body, "<h2")
	if i < 0 {
		return body, ""
	}
	return body[:i], body[i:]
}
