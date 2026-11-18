package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

const (
	PortMin = 45690
	PortMax = 45699
)

// Server is the smallest localhost surface Monaco needs, plus /api/health
// so a running process can be checked. Snippets, config, and windows are native.
type Server struct {
	WebRoot string
	Version string
	mux     *http.ServeMux
}

func New(webRoot, version string) *Server {
	s := &Server{WebRoot: webRoot, Version: version, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /editor", s.editor)
	if webRoot != "" {
		monaco := filepath.Join(webRoot, "vendor", "monaco")
		s.mux.Handle("GET /web/vendor/monaco/", http.StripPrefix("/web/vendor/monaco/", http.FileServer(http.Dir(monaco))))
	}
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func Listen() (net.Listener, int, error) {
	var last error
	for p := PortMin; p <= PortMax; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			return ln, p, nil
		}
		last = err
	}
	return nil, 0, fmt.Errorf("ports %d-%d busy: %w", PortMin, PortMax, last)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	port := 0
	if _, p, err := net.SplitHostPort(r.Host); err == nil {
		fmt.Sscanf(p, "%d", &port)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "port": port, "version": s.Version})
}

func (s *Server) editor(w http.ResponseWriter, r *http.Request) {
	if s.WebRoot == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, filepath.Join(s.WebRoot, "editor.html"))
}

func FindWebRoot() string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "web"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "web"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}
