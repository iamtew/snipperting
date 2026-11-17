package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"snipperting/internal/config"
	"snipperting/internal/snippet"
)

func setup(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := snippet.Encode(snippet.Meta{Label: "Demo", Action: "clipboard", Enabled: true, About: "hi"}, "hello body")
	if err := os.WriteFile(filepath.Join(data, "demo.md"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := config.LoadAt(filepath.Join(dir, "config.json"), data)
	if err != nil {
		t.Fatal(err)
	}
	var clip string
	s := New(store, "", "# manual\n", "0.0.1", Hooks{
		Clipboard: func(text string) error { clip = text; return nil },
		Inject:    func(text string) error { clip = "INJECT:" + text; return nil },
	})
	t.Cleanup(func() { _ = clip })
	return s, data
}

func TestHealth(t *testing.T) {
	s, _ := setup(t)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "http://127.0.0.1:45690/api/health", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true || got["version"] != "0.0.1" {
		t.Fatalf("%v", got)
	}
}

func TestSnippetsActAndFile(t *testing.T) {
	s, data := setup(t)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "/api/snippets", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}

	rr = httptest.NewRecorder()
	body := bytes.NewBufferString(`{"path":"demo.md","op":"clipboard"}`)
	s.ServeHTTP(rr, httptest.NewRequest("POST", "/api/snippets/act", body))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}

	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("POST", "/api/snippets/act", bytes.NewBufferString(`{"path":"demo.md","op":"toggle-state"}`)))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	raw, _ := os.ReadFile(filepath.Join(data, "demo.md"))
	if snippet.Parse(raw).Meta.Enabled {
		t.Fatal("expected disabled")
	}

	rr = httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/file?path=demo.md", bytes.NewBufferString("----\nlabel: Z\n----\nnew\n"))
	s.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}

	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "/api/file?path=demo.md", nil))
	if !bytes.Contains(rr.Body.Bytes(), []byte("new")) {
		t.Fatal(rr.Body.String())
	}

	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("PUT", "/api/file?path=../x.md", bytes.NewBufferString("nope")))
	if rr.Code == 200 {
		t.Fatal("traversal")
	}

	on := false
	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("POST", "/api/bulk", bytes.NewBufferString(`{"items":[{"path":"demo.md","action":"inject","enabled":true,"hidden":true}]}`)))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	raw, _ = os.ReadFile(filepath.Join(data, "demo.md"))
	m := snippet.Parse(raw).Meta
	if m.Action != "inject" || !m.Enabled || !m.Hidden {
		t.Fatalf("%+v", m)
	}
	_ = on
}

func TestListenRange(t *testing.T) {
	ln, port, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if port < PortMin || port > PortMax {
		t.Fatalf("port %d", port)
	}
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))
}
