package winapp

import (
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
	if uiPal.bgBrush == 0 || uiPal.fieldBrush == 0 || uiPal.accBrush == 0 {
		t.Fatal("palette brushes")
	}
	if messageFont() == 0 {
		t.Fatal("messageFont")
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
