package docs

import "embed"

//go:embed shots/*.png
var shotFS embed.FS

// ShotPNGs is shots/editor.png and shots/preferences.png for Help data URIs.
func ShotPNGs() map[string][]byte {
	out := map[string][]byte{}
	for _, name := range []string{"editor.png", "preferences.png"} {
		b, err := shotFS.ReadFile("shots/" + name)
		if err != nil {
			continue
		}
		out["shots/"+name] = b
	}
	return out
}
