package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"snipperting/docs"
	"snipperting/internal/config"
	"snipperting/internal/winapp"
)

var version = "dev"

func main() {
	if !winapp.SingleInstance("Local\\snipperting") {
		os.Exit(0)
	}
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	winapp.StartUI(cfg, docs.Manual, version, vcsRev())
	winapp.RunTray(loadIcon(), cfg)
}

func vcsRev() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	rev, dirty := "", false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return ""
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if dirty {
		return rev + "-dirty"
	}
	return rev
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
