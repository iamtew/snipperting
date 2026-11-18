package winapp

import (
	"bytes"
	"image/png"
	"path/filepath"
	"testing"
	"unsafe"

	"snipperting/internal/config"
)

func TestThemeLayout(t *testing.T) {
	if n := unsafe.Sizeof(actctx{}); n != 56 {
		t.Fatalf("actctx size %d", n)
	}
	if n := unsafe.Sizeof(nonClientMetrics{}); n != 504 {
		t.Fatalf("ncm size %d", n)
	}
	if n := unsafe.Sizeof(drawItem{}); n != 64 {
		t.Fatalf("drawItem size %d", n)
	}
	loadPalette()
	if uiPal.bgBrush == 0 || uiPal.fieldBrush == 0 || uiPal.accBrush == 0 || uiPal.ruleBrush == 0 {
		t.Fatal("palette brushes")
	}
	if messageFont() == 0 {
		t.Fatal("messageFont")
	}
}

func TestChromeIconsDecode(t *testing.T) {
	for _, name := range []string{"folder", "file", "file-plus", "folder-plus", "device-floppy", "bold", "italic", "strikethrough", "heading", "quote", "list", "list-numbers", "code", "link", "markdown", "trash"} {
		raw := chromeIcon(name)
		if len(raw) <= 22 {
			t.Fatalf("%s empty", name)
		}
		if _, err := png.Decode(bytes.NewReader(raw[22:])); err != nil {
			t.Fatalf("%s png: %v", name, err)
		}
	}
}

func TestWantDarkHonorsConfig(t *testing.T) {
	dir := t.TempDir()
	s, err := config.LoadAt(filepath.Join(dir, "c.json"), dir)
	if err != nil {
		t.Fatal(err)
	}
	uiCfg = s
	t.Cleanup(func() { uiCfg = nil })
	_ = s.Set(config.File{DataDir: dir, Theme: "dark", ClipNotice: true})
	if !wantDark() {
		t.Fatal("dark")
	}
	_ = s.Set(config.File{DataDir: dir, Theme: "light", ClipNotice: true})
	if wantDark() {
		t.Fatal("light")
	}
}

func TestLoadPaletteBrand(t *testing.T) {
	dir := t.TempDir()
	s, err := config.LoadAt(filepath.Join(dir, "c.json"), dir)
	if err != nil {
		t.Fatal(err)
	}
	uiCfg = s
	t.Cleanup(func() { uiCfg = nil })
	navy, sky, teal := rgb(0x02, 0x30, 0x47), rgb(0x8e, 0xca, 0xe6), rgb(0x21, 0x9e, 0xbc)
	_ = s.Set(config.File{DataDir: dir, Theme: "light", ClipNotice: true})
	loadPalette()
	if uiPal.bg != sky || uiPal.field != sky || uiPal.rule != teal || uiPal.text != navy || uiPal.accent != rgb(0xfb, 0x85, 0x00) || uiPal.accentText != navy {
		t.Fatalf("light %#v", uiPal)
	}
	_ = s.Set(config.File{DataDir: dir, Theme: "dark", ClipNotice: true})
	loadPalette()
	if uiPal.bg != navy || uiPal.field != navy || uiPal.rule != teal || uiPal.text != sky || uiPal.accent != rgb(0xff, 0xb7, 0x03) || uiPal.accentText != navy {
		t.Fatalf("dark %#v", uiPal)
	}
}
