package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const Version = "0.0.1"

type File struct {
	DataDir string `json:"dataDir"`
	Theme   string `json:"theme"`
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
	} else if err := json.Unmarshal(b, &s.cur); err != nil {
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

func LoadAt(configPath, dataDir string) (*Store, error) {
	s := &Store{path: configPath, cur: File{DataDir: dataDir, Theme: "system"}}
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
	s.cur = next
	if err := s.normalize(); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.cur, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(b, '\n'), 0o644)
}
