package winapp

import (
	"os"

	"github.com/energye/systray"

	"snipperting/internal/config"
	"snipperting/internal/snippet"
)

func RunTray(port int, icon []byte, cfg *config.Store, onExit func()) {
	_ = port
	systray.Run(func() {
		if len(icon) > 0 {
			systray.SetIcon(icon)
		}
		systray.SetTitle("snipperting")
		systray.SetTooltip("snipperting")
		fillMenu(cfg)
		openNative := func(m systray.IMenu) {
			RememberForeground()
			fillMenu(cfg)
			_ = m.ShowMenu()
		}
		systray.SetOnClick(openNative)
		systray.SetOnRClick(openNative)
	}, func() {
		if onExit != nil {
			onExit()
		}
		os.Exit(0)
	})
}

func fillMenu(cfg *config.Store) {
	systray.ResetMenu()
	dataDir := cfg.Get().DataDir
	tree, err := snippet.Walk(dataDir)
	if err == nil {
		addNodes(nil, dataDir, tree.Children)
	}
	systray.AddSeparator()
	addOpen("Edit", "editor")
	addOpen("Preferences", "prefs")
	addOpen("Help", "help")
	addOpen("About", "about")
	quit := systray.AddMenuItem("Exit", "Quit snipperting")
	quit.Click(func() { systray.Quit() })
}

func addOpen(title, kind string) {
	it := systray.AddMenuItem(title, "")
	switch kind {
	case "editor":
		it.SetIcon(menuIcon("pencil"))
	case "prefs":
		it.SetIcon(menuIcon("settings"))
	case "help":
		it.SetIcon(menuIcon("help"))
	}
	it.Click(func() { Open(kind, "") })
}

func addNodes(parent *systray.MenuItem, dataDir string, nodes []snippet.Node) {
	for _, n := range nodes {
		n := n
		if n.Type == "file" && n.Hidden {
			continue
		}
		title := n.Label
		if n.Type == "dir" {
			title = n.Name
		}
		var it *systray.MenuItem
		if parent == nil {
			it = systray.AddMenuItem(title, n.About)
		} else {
			it = parent.AddSubMenuItem(title, n.About)
		}
		if n.Type == "dir" {
			it.SetIcon(menuIcon("folder"))
			addNodes(it, dataDir, n.Children)
			continue
		}
		icon := "clipboard"
		if n.Action == "inject" {
			icon = "keyboard"
		}
		it.SetIcon(menuIcon(icon))
		if !n.Enabled {
			it.Disable()
			continue
		}
		path, action := n.Path, n.Action
		it.Click(func() { fireSnippet(dataDir, path, action) })
	}
}

func fireSnippet(dataDir, rel, action string) {
	raw, err := snippet.Read(dataDir, rel)
	if err != nil {
		return
	}
	body := snippet.Parse(raw).Body
	if action == "inject" {
		_ = Inject(body)
		return
	}
	_ = SetClipboard(body)
	NotifyCopied()
}
