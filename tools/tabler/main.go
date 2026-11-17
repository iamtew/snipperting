package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func main() {
	in := "web/vendor/tabler"
	out := "internal/winapp/icons"
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}
	ents, err := os.ReadDir(in)
	if err != nil {
		fatal(err)
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".svg") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(in, e.Name()))
		if err != nil {
			fatal(err)
		}
		svg := strings.ReplaceAll(string(b), "currentColor", "#111111")
		icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
		if err != nil {
			fatal(err)
		}
		const s = 16
		icon.SetTarget(0, 0, s, s)
		img := image.NewRGBA(image.Rect(0, 0, s, s))
		scan := rasterx.NewScannerGV(s, s, img, img.Bounds())
		icon.Draw(rasterx.NewDasher(s, s, scan), 1)
		var pngBuf bytes.Buffer
		if err := png.Encode(&pngBuf, img); err != nil {
			fatal(err)
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) + ".ico"
		if err := os.WriteFile(filepath.Join(out, name), icoPNG(pngBuf.Bytes(), s), 0o644); err != nil {
			fatal(err)
		}
		n++
	}
	fmt.Printf("wrote %d icons into %s\n", n, out)
}

func icoPNG(pngBytes []byte, size byte) []byte {
	var ico bytes.Buffer
	_ = binary.Write(&ico, binary.LittleEndian, uint16(0))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	ico.WriteByte(size)
	ico.WriteByte(size)
	ico.WriteByte(0)
	ico.WriteByte(0)
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(32))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(len(pngBytes)))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(22))
	ico.Write(pngBytes)
	return ico.Bytes()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
