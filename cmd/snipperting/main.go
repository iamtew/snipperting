package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

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
	winapp.StartUI(cfg, docs.Manual, docs.ShotPNGs(), version, vcsRev())
	prefs, edit, editFile := parseOpen(os.Args[1:])
	if prefs {
		winapp.Open("prefs", "")
	}
	if edit {
		winapp.Open("editor", editFile)
	}
	winapp.RunTray(loadIcon(), cfg)
}

// parseOpen reads --prefs and --edit. A following arg that does not start with - is the note path.
func parseOpen(args []string) (prefs, edit bool, editFile string) {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--prefs":
			prefs = true
		case args[i] == "--edit":
			edit = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				editFile = args[i]
			}
		case strings.HasPrefix(args[i], "--edit="):
			edit = true
			editFile = strings.TrimPrefix(args[i], "--edit=")
		}
	}
	return
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
	winapp.Alert(err.Error())
	os.Exit(1)
}
