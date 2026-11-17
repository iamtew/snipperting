package winapp

import (
	"testing"
	"unsafe"
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
