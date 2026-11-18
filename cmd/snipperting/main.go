package main

import (
	"fmt"
	"os"
	"path/filepath"

	"snipperting/docs"
	"snipperting/internal/config"
	"snipperting/internal/winapp"
)

var version = config.Version

func main() {
	if !winapp.SingleInstance("Local\\snipperting") {
		os.Exit(0)
	}
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	winapp.StartUI(cfg, docs.Manual, version)
	winapp.RunTray(loadIcon(), cfg)
}

func loadIcon() []byte {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "snipperting.ico"), filepath.Join(dir, "build", "snipperting.ico"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "build", "snipperting.ico"))
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			return b
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
