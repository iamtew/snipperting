package winapp

import (
	"os"
	"strings"
	"time"

	"github.com/energye/systray"

	"snipperting/internal/config"
	"snipperting/internal/snippet"
)

func RunTray(icon []byte, cfg *config.Store) {
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
		os.Exit(0)
	})
}

func fillMenu(cfg *config.Store) {
	systray.ResetMenu()
	dataDir := cfg.Get().DataDir
	tree, err := snippet.Walk(dataDir)
	if err == nil {
		snippet.OrderChildren(tree.Children, false)
		addNodes(nil, dataDir, tree.Children)
	}
	systray.AddSeparator()
	addNew(cfg)
	addOpen("Edit", "editor")
	addOpen("Preferences", "prefs")
	addOpen("Help", "help")
	addOpen("About", "about")
	quit := systray.AddMenuItem("Exit", "Quit snipperting")
	quit.SetIcon(menuIcon("logout"))
	quit.Click(func() { systray.Quit() })
}

func addNew(cfg *config.Store) {
	it := systray.AddMenuItem("New", "")
	it.SetIcon(menuIcon("file-plus"))
	it.Click(func() { trayNew(cfg) })
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
	case "about":
		it.SetIcon(menuIcon("info-circle"))
	}
	it.Click(func() { Open(kind, "") })
}

func noteStampName(t time.Time) string {
	return "note-" + t.Format("0102-1504") + ".md"
}

func uniqueRootName(dataDir, base string) string {
	taken := map[string]bool{}
	ents, err := os.ReadDir(dataDir)
	if err == nil {
		for _, e := range ents {
			taken[strings.ToLower(e.Name())] = true
		}
	}
	return uniqueLeaf(taken, base)
}

func trayNew(cfg *config.Store) {
	dataDir := cfg.Get().DataDir
	name := uniqueRootName(dataDir, noteStampName(time.Now()))
	body := GetClipboard()
	if body == "" {
		body = "\n"
	}
	if err := snippet.CreateFile(dataDir, name, body); err != nil {
		Alert("Can't create that note.")
		return
	}
	queueOpen(openReq{kind: "editor", file: name, bodyEnd: true})
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
		for _, a := range snippet.ParseActions(n.Action) {
			if a == "inject" {
				icon = "keyboard"
				break
			}
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
		Alert("Can't read that note.")
		return
	}
	body := snippet.Parse(raw).Body
	copied := false
	for _, a := range snippet.ParseActions(action) {
		if a == "inject" {
			if err := Inject(body); err != nil {
				Alert("Can't inject that note.")
			}
			continue
		}
		if err := SetClipboard(body); err != nil {
			Alert("Can't copy that note.")
			continue
		}
		copied = true
	}
	if copied {
		NotifyCopied()
	}
}
