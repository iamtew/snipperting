package winapp

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/energye/systray"

	"snipperting/internal/config"
	"snipperting/internal/snippet"
)

var (
	menuMu      sync.Mutex
	lastMenuSig string
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
		ReleaseInstance()
		os.Exit(0)
	})
}

func fillMenu(cfg *config.Store) {
	menuMu.Lock()
	defer menuMu.Unlock()
	dataDir := cfg.Get().DataDir
	tree, err := snippet.Walk(dataDir)
	if err != nil {
		lastMenuSig = ""
		systray.ResetMenu()
		bad := systray.AddMenuItem("Can't read snippets", "")
		bad.Disable()
		addMenuFooter(cfg)
		return
	}
	snippet.OrderChildren(tree.Children, false)
	sig := dataDir + "\n" + nodeSig(tree)
	if sig == lastMenuSig {
		return
	}
	lastMenuSig = sig
	systray.ResetMenu()
	addNodes(nil, dataDir, tree.Children)
	addMenuFooter(cfg)
}

func addMenuFooter(cfg *config.Store) {
	systray.AddSeparator()
	addNew(cfg)
	addOpen("Edit", "editor")
	addOpen("Preferences", "prefs")
	addOpen("Help", "help")
	addOpen("About", "about")
	quit := systray.AddMenuItem("Exit", "Quit snipperting")
	quit.SetIcon(chromeIcon("logout"))
	quit.Click(func() { systray.Quit() })
}

func nodeSig(n snippet.Node) string {
	var b strings.Builder
	appendNodeSig(&b, n)
	return b.String()
}

func appendNodeSig(b *strings.Builder, n snippet.Node) {
	b.WriteString(n.Type)
	b.WriteByte(0)
	b.WriteString(n.Path)
	b.WriteByte(0)
	b.WriteString(n.Name)
	b.WriteByte(0)
	b.WriteString(n.Label)
	b.WriteByte(0)
	b.WriteString(n.About)
	b.WriteByte(0)
	b.WriteString(n.Action)
	b.WriteByte(0)
	b.WriteString(strconv.FormatBool(n.Enabled))
	b.WriteByte(0)
	b.WriteString(strconv.FormatBool(n.Hidden))
	b.WriteByte('\n')
	for _, c := range n.Children {
		appendNodeSig(b, c)
	}
}

func addNew(cfg *config.Store) {
	it := systray.AddMenuItem("New", "")
	it.SetIcon(chromeIcon("file-plus"))
	it.Click(func() { trayNew(cfg) })
}

func addOpen(title, kind string) {
	it := systray.AddMenuItem(title, "")
	switch kind {
	case "editor":
		it.SetIcon(chromeIcon("pencil"))
	case "prefs":
		it.SetIcon(chromeIcon("settings"))
	case "help":
		it.SetIcon(chromeIcon("help"))
	case "about":
		it.SetIcon(chromeIcon("info-circle"))
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
			it.SetIcon(chromeIcon("folder"))
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
		it.SetIcon(chromeIcon(icon))
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
		cancelClipRestore()
		NotifyCopied()
	}
}
