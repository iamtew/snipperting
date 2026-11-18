package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type File struct {
	DataDir    string `json:"dataDir"`
	Theme      string `json:"theme"`
	ClipNotice bool   `json:"clipNotice"`
}

type Store struct {
	mu   sync.Mutex
	path string
	cur  File
}

func Path() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "config.json"), nil
}

func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".snipperting"), nil
}

func Load() (*Store, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	s := &Store{path: p}
	missing := false
	b, err := os.ReadFile(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		missing = true
		s.cur.ClipNotice = true
	} else if err := decodeFile(b, &s.cur); err != nil {
		return nil, err
	}
	if err := s.normalize(); err != nil {
		return nil, err
	}
	if missing {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func decodeFile(b []byte, cur *File) error {
	var d struct {
		DataDir    string `json:"dataDir"`
		Theme      string `json:"theme"`
		ClipNotice *bool  `json:"clipNotice"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	cur.DataDir = d.DataDir
	cur.Theme = d.Theme
	if d.ClipNotice == nil {
		cur.ClipNotice = true
	} else {
		cur.ClipNotice = *d.ClipNotice
	}
	return nil
}

func LoadAt(configPath, dataDir string) (*Store, error) {
	s := &Store{path: configPath, cur: File{DataDir: dataDir, Theme: "system", ClipNotice: true}}
	if err := s.normalize(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) normalize() error {
	if s.cur.Theme != "light" && s.cur.Theme != "dark" {
		s.cur.Theme = "system"
	}
	if s.cur.DataDir == "" {
		d, err := DefaultDataDir()
		if err != nil {
			return err
		}
		s.cur.DataDir = d
	}
	return os.MkdirAll(s.cur.DataDir, 0o755)
}

func (s *Store) Get() File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

func (s *Store) Set(next File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.cur
	s.cur = next
	err := s.normalize()
	if err == nil {
		err = s.saveLocked()
	}
	if err != nil {
		s.cur = prev
		return err
	}
	return nil
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.cur, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	f, err := os.CreateTemp(dir, ".cfg-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(append(b, '\n'))
	cerr := f.Close()
	if werr != nil {
		os.Remove(tmp)
		return werr
	}
	if cerr != nil {
		os.Remove(tmp)
		return cerr
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
