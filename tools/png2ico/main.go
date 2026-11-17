package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	_ "image/png"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: png2ico in.png out.ico")
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatal(err)
	}
	f, err := os.Open(in)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		fatal(err)
	}
	sized := resizeNN(img, 32, 32)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, sized); err != nil {
		fatal(err)
	}
	payload := pngBuf.Bytes()
	var ico bytes.Buffer
	_ = binary.Write(&ico, binary.LittleEndian, uint16(0))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	ico.WriteByte(32)
	ico.WriteByte(32)
	ico.WriteByte(0)
	ico.WriteByte(0)
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(32))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(len(payload)))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(6+16))
	ico.Write(payload)
	if err := os.WriteFile(out, ico.Bytes(), 0o644); err != nil {
		fatal(err)
	}
}

func resizeNN(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*sw/w
			sy := sb.Min.Y + y*sh/h
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
