package snippet

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

const fence = "----"

type Meta struct {
	Label   string
	About   string
	Action  string // clipboard | inject
	Enabled bool
	Hidden  bool
}

type Doc struct {
	Meta Meta
	Body string
}

type Node struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	About    string `json:"about,omitempty"`
	Action   string `json:"action,omitempty"`
	Enabled  bool   `json:"enabled"`
	Hidden   bool   `json:"hidden,omitempty"`
	Children []Node `json:"children,omitempty"`
}

func Parse(raw []byte) Doc {
	text := string(raw)
	text = strings.TrimPrefix(text, "\ufeff")
	lines := splitKeep(text)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != fence {
		return Doc{Meta: defaults(), Body: strings.TrimPrefix(text, "\r\n")}
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == fence {
			end = i
			break
		}
	}
	if end < 0 {
		return Doc{Meta: defaults(), Body: text}
	}
	m := defaults()
	for _, line := range lines[1:end] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		applyKV(&m, strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v))
	}
	body := strings.Join(lines[end+1:], "\n")
	body = strings.TrimPrefix(body, "\n")
	return Doc{Meta: m, Body: body}
}

func Encode(m Meta, body string) []byte {
	if m.Action != "inject" {
		m.Action = "clipboard"
	}
	state := "disable"
	if m.Enabled {
		state = "enable"
	}
	hidden := "false"
	if m.Hidden {
		hidden = "true"
	}
	var b bytes.Buffer
	b.WriteString(fence + "\n")
	b.WriteString("label: " + m.Label + "\n")
	b.WriteString("about: " + m.About + "\n")
	b.WriteString("action: " + m.Action + "\n")
	b.WriteString("state: " + state + "\n")
	b.WriteString("hidden: " + hidden + "\n")
	b.WriteString(fence + "\n")
	b.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func ApplyMeta(raw []byte, edit func(*Meta)) []byte {
	d := Parse(raw)
	edit(&d.Meta)
	return Encode(d.Meta, d.Body)
}

func defaults() Meta {
	return Meta{Action: "clipboard", Enabled: true}
}

func applyKV(m *Meta, k, v string) {
	switch k {
	case "label":
		m.Label = v
	case "about":
		m.About = v
	case "action":
		if strings.EqualFold(v, "inject") {
			m.Action = "inject"
		} else {
			m.Action = "clipboard"
		}
	case "state":
		m.Enabled = parseOn(v, true)
	case "hidden":
		m.Hidden = parseOn(v, false)
	}
}

// parseOn accepts enable/disable, on/off, true/false. defaultOn if empty/unknown.
func parseOn(v string, defaultOn bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "enable", "enabled", "on", "true", "yes", "1":
		return true
	case "disable", "disabled", "off", "false", "no", "0":
		return false
	default:
		return defaultOn
	}
}

func splitKeep(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func Walk(dataDir string) (Node, error) {
	root := Node{Type: "dir", Path: "", Name: "", Enabled: true}
	err := walkDir(dataDir, "", &root)
	return root, err
}

func walkDir(root, rel string, n *Node) error {
	dir := filepath.Join(root, filepath.FromSlash(rel))
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range ents {
		name := e.Name()
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if e.IsDir() {
			ch := Node{Type: "dir", Path: childRel, Name: name, Enabled: true}
			if err := walkDir(root, childRel, &ch); err != nil {
				return err
			}
			if len(ch.Children) > 0 {
				n.Children = append(n.Children, ch)
			}
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		d := Parse(raw)
		label := d.Meta.Label
		if label == "" {
			label = strings.TrimSuffix(name, filepath.Ext(name))
		}
		n.Children = append(n.Children, Node{
			Type:    "file",
			Path:    childRel,
			Name:    name,
			Label:   label,
			About:   d.Meta.About,
			Action:  d.Meta.Action,
			Enabled: d.Meta.Enabled,
			Hidden:  d.Meta.Hidden,
		})
	}
	return nil
}

func JoinUnder(dataDir, rel string) (string, error) {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || filepath.IsAbs(filepath.FromSlash(rel)) || strings.Contains(rel, ":") {
		return "", os.ErrInvalid
	}
	base, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	full := filepath.Join(base, filepath.FromSlash(rel))
	full, err = filepath.Abs(full)
	if err != nil {
		return "", err
	}
	sep := string(os.PathSeparator)
	if full != base && !strings.HasPrefix(strings.ToLower(full), strings.ToLower(base)+sep) {
		return "", os.ErrInvalid
	}
	return full, nil
}

func Read(dataDir, rel string) ([]byte, error) {
	p, err := JoinUnder(dataDir, rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func Write(dataDir, rel string, data []byte) error {
	p, err := JoinUnder(dataDir, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func UpdateMeta(dataDir, rel string, edit func(*Meta)) error {
	raw, err := Read(dataDir, rel)
	if err != nil {
		return err
	}
	return Write(dataDir, rel, ApplyMeta(raw, edit))
}
