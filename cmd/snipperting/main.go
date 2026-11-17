package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"snipperting/docs"
	"snipperting/internal/config"
	"snipperting/internal/httpapi"
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
	web := httpapi.FindWebRoot()
	hooks := httpapi.Hooks{
		Clipboard: winapp.SetClipboard,
		Inject:    winapp.Inject,
		OpenUI: func(kind, file string) {
			base := fmt.Sprintf("http://127.0.0.1:%d", listenPort)
			switch kind {
			case "editor":
				url := base + "/editor"
				if file != "" {
					url += "?path=" + file
				}
				winapp.OpenWindow("snipperting editor", url, 1100, 720)
			case "prefs":
				winapp.OpenWindow("snipperting preferences", base+"/prefs", 720, 560)
			case "help":
				winapp.OpenWindow("snipperting help", base+"/help", 720, 640)
			case "about":
				winapp.OpenWindow("snipperting about", base+"/about", 420, 360)
			}
		},
		Exit: func() {
			os.Exit(0)
		},
	}
	srv := httpapi.New(cfg, web, docs.Manual, version, hooks)
	ln, port, err := httpapi.Listen()
	if err != nil {
		fatal(err)
	}
	listenPort = port
	go func() {
		if err := http.Serve(ln, srv); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	icon := loadIcon()
	winapp.RunTray(port, icon, cfg, func() { _ = ln.Close() })
}

var listenPort int

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
