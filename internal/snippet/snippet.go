package snippet

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
)

const fence = "----"

type Meta struct {
	Label   string
	About   string
	Action  string // clipboard and/or inject, comma-separated
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
	m.Action = FormatActions(ParseActions(m.Action))
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

// ParseActions splits a header action value. Known tokens: inject, clipboard.
// Unknown dropped, duplicates collapsed (first-seen). Empty → clipboard.
func ParseActions(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "inject" && p != "clipboard" {
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"clipboard"}
	}
	return out
}

func FormatActions(acts []string) string {
	return strings.Join(ParseActions(strings.Join(acts, ",")), ", ")
}

func applyKV(m *Meta, k, v string) {
	switch k {
	case "label":
		m.Label = v
	case "about":
		m.About = v
	case "action":
		m.Action = FormatActions(ParseActions(v))
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

// OrderChildren stable-partitions files vs dirs at every level.
// dirsFirst true: folders then files (Edit tree). false: files then folders (tray, Bulk Edit).
func OrderChildren(nodes []Node, dirsFirst bool) {
	if len(nodes) == 0 {
		return
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		di := nodes[i].Type == "dir"
		dj := nodes[j].Type == "dir"
		if di == dj {
			return false
		}
		if dirsFirst {
			return di
		}
		return !di
	})
	for i := range nodes {
		OrderChildren(nodes[i].Children, dirsFirst)
	}
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
			n.Children = append(n.Children, ch)
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
	clean := path.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", os.ErrInvalid
	}
	for _, p := range strings.Split(clean, "/") {
		if p == ".." {
			return "", os.ErrInvalid
		}
	}
	base, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	full := filepath.Join(base, filepath.FromSlash(clean))
	full, err = filepath.Abs(full)
	if err != nil {
		return "", err
	}
	sep := string(os.PathSeparator)
	baseLow := strings.ToLower(base)
	fullLow := strings.ToLower(full)
	if fullLow == baseLow || !strings.HasPrefix(fullLow, baseLow+sep) {
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

func CreateDir(dataDir, rel string) error {
	p, err := JoinUnder(dataDir, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	return nil
}

func Remove(dataDir, rel string) error {
	p, err := JoinUnder(dataDir, rel)
	if err != nil {
		return err
	}
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return os.RemoveAll(p)
	}
	return os.Remove(p)
}

func Move(dataDir, srcRel, destRel string) error {
	srcRel = filepath.ToSlash(strings.TrimPrefix(srcRel, "/"))
	destRel = filepath.ToSlash(strings.TrimPrefix(destRel, "/"))
	if srcRel == "" || destRel == "" {
		return os.ErrInvalid
	}
	src, err := JoinUnder(dataDir, srcRel)
	if err != nil {
		return err
	}
	dest, err := JoinUnder(dataDir, destRel)
	if err != nil {
		return err
	}
	sep := string(os.PathSeparator)
	srcLow := strings.ToLower(src)
	destLow := strings.ToLower(dest)
	if destLow == srcLow || strings.HasPrefix(destLow, srcLow+sep) {
		return os.ErrInvalid
	}
	if _, err := os.Stat(dest); err == nil {
		return os.ErrExist
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	from, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(dest)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, 0); err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return os.ErrExist
		}
		return err
	}
	return nil
}

func CreateFile(dataDir, rel, body string) error {
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		rel = rel + ".md"
	}
	p, err := JoinUnder(dataDir, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	_, werr := f.Write(Encode(defaults(), body))
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}
