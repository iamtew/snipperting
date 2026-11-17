package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"snipperting/internal/config"
	"snipperting/internal/snippet"
)

const (
	PortMin = 45690
	PortMax = 45699
)

type Hooks struct {
	Clipboard func(text string) error
	Inject    func(text string) error
	OpenUI    func(kind, file string)
	Exit      func()
}

type Server struct {
	Cfg     *config.Store
	WebRoot string
	Manual  string
	Version string
	Hooks   Hooks
	mux     *http.ServeMux
}

func New(cfg *config.Store, webRoot, manual, version string, hooks Hooks) *Server {
	s := &Server{Cfg: cfg, WebRoot: webRoot, Manual: manual, Version: version, Hooks: hooks, mux: http.NewServeMux()}
	s.routes()
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

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/config", s.getConfig)
	s.mux.HandleFunc("PUT /api/config", s.putConfig)
	s.mux.HandleFunc("GET /api/snippets", s.getSnippets)
	s.mux.HandleFunc("POST /api/snippets/act", s.act)
	s.mux.HandleFunc("GET /api/file", s.getFile)
	s.mux.HandleFunc("PUT /api/file", s.putFile)
	s.mux.HandleFunc("POST /api/bulk", s.bulk)
	s.mux.HandleFunc("GET /api/manual", s.getManual)
	s.mux.HandleFunc("POST /api/ui", s.openUI)
	s.mux.HandleFunc("GET /manual", s.getManual)
	s.mux.HandleFunc("GET /menu", s.page("menu.html"))
	s.mux.HandleFunc("GET /editor", s.page("editor.html"))
	s.mux.HandleFunc("GET /prefs", s.page("prefs.html"))
	s.mux.HandleFunc("GET /help", s.page("help.html"))
	s.mux.HandleFunc("GET /about", s.page("about.html"))
	if s.WebRoot != "" {
		s.mux.Handle("GET /web/", http.StripPrefix("/web/", http.FileServer(http.Dir(s.WebRoot))))
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	port := 0
	if h, _, err := net.SplitHostPort(r.Host); err == nil && h != "" {
		_, _ = h, port
	}
	if _, p, err := net.SplitHostPort(r.Host); err == nil {
		fmt.Sscanf(p, "%d", &port)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "port": port, "version": s.Version})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Cfg.Get())
}

func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var next config.File
	if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Cfg.Set(next); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.Cfg.Get())
}

func (s *Server) getSnippets(w http.ResponseWriter, _ *http.Request) {
	tree, err := snippet.Walk(s.Cfg.Get().DataDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

type actReq struct {
	Path  string `json:"path"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

func (s *Server) act(w http.ResponseWriter, r *http.Request) {
	var req actReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dir := s.Cfg.Get().DataDir
	raw, err := snippet.Read(dir, req.Path)
	if err != nil {
		status := http.StatusBadRequest
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	doc := snippet.Parse(raw)
	switch req.Op {
	case "clipboard":
		if s.Hooks.Clipboard == nil {
			http.Error(w, "clipboard not available", http.StatusNotImplemented)
			return
		}
		if err := s.Hooks.Clipboard(doc.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "inject":
		if s.Hooks.Inject == nil {
			http.Error(w, "inject not available", http.StatusNotImplemented)
			return
		}
		if err := s.Hooks.Inject(doc.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "toggle-state":
		err = snippet.UpdateMeta(dir, req.Path, func(m *snippet.Meta) { m.Enabled = !m.Enabled })
	case "set-action":
		err = snippet.UpdateMeta(dir, req.Path, func(m *snippet.Meta) {
			if req.Value == "inject" {
				m.Action = "inject"
			} else {
				m.Action = "clipboard"
			}
		})
	case "set-hidden":
		err = snippet.UpdateMeta(dir, req.Path, func(m *snippet.Meta) {
			m.Hidden = req.Value == "true" || req.Value == "on" || req.Value == "1"
		})
	default:
		http.Error(w, "unknown op", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	raw, err := snippet.Read(s.Cfg.Get().DataDir, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": p, "content": string(raw)})
}

func (s *Server) putFile(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := snippet.Write(s.Cfg.Get().DataDir, p, b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type bulkReq struct {
	Items []bulkItem `json:"items"`
}

type bulkItem struct {
	Path    string `json:"path"`
	Action  string `json:"action"`
	Enabled *bool  `json:"enabled"`
	Hidden  *bool  `json:"hidden"`
}

func (s *Server) bulk(w http.ResponseWriter, r *http.Request) {
	var req bulkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dir := s.Cfg.Get().DataDir
	for _, it := range req.Items {
		it := it
		if err := snippet.UpdateMeta(dir, it.Path, func(m *snippet.Meta) {
			if it.Action == "inject" || it.Action == "clipboard" {
				m.Action = it.Action
			}
			if it.Enabled != nil {
				m.Enabled = *it.Enabled
			}
			if it.Hidden != nil {
				m.Hidden = *it.Hidden
			}
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) getManual(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = io.WriteString(w, s.Manual)
}

type uiReq struct {
	Kind string `json:"kind"`
	File string `json:"file"`
}

func (s *Server) openUI(w http.ResponseWriter, r *http.Request) {
	var req uiReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Kind == "exit" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		if s.Hooks.Exit != nil {
			go s.Hooks.Exit()
		}
		return
	}
	if s.Hooks.OpenUI != nil {
		go s.Hooks.OpenUI(req.Kind, req.File)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.WebRoot == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.WebRoot, name))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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
