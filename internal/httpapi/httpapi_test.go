package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthOnly(t *testing.T) {
	s := New("", "0.0.1")
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
	for _, p := range []string{"/api/config", "/api/snippets", "/api/file", "/prefs", "/help", "/about"} {
		rr = httptest.NewRecorder()
		s.ServeHTTP(rr, httptest.NewRequest("GET", p, nil))
		if rr.Code == 200 {
			t.Fatalf("%s still served", p)
		}
	}
}

func TestMonacoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "vendor", "monaco"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "editor.html"), []byte("EDITOR"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "monaco", "loader.js"), []byte("LOADER"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(dir, "0.0.1")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "/editor", nil))
	if rr.Body.String() != "EDITOR" {
		t.Fatalf("editor %q", rr.Body.String())
	}
	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "/web/vendor/monaco/loader.js", nil))
	if rr.Body.String() != "LOADER" {
		t.Fatalf("loader %q code %d", rr.Body.String(), rr.Code)
	}
	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, httptest.NewRequest("GET", "/web/secret.txt", nil))
	if rr.Code == 200 {
		t.Fatal("web root is too open")
	}
}

func TestEditorHTMLHasWorker(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "editor.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("workerMain.js")) {
		t.Fatal("editor.html must pin Monaco workers")
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

func TestListenRange(t *testing.T) {
	ln, port, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if port < PortMin || port > PortMax {
		t.Fatalf("port %d", port)
	}
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
}
