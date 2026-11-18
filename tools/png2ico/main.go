package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func main() {
	if len(os.Args) != 3 {
		usage()
	}
	in, out := os.Args[1], os.Args[2]
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatal(err)
	}
	var err error
	switch strings.ToLower(filepath.Ext(out)) {
	case ".png":
		err = writePNG(in, out, 64)
	case ".ico":
		err = writeICO(in, out)
	default:
		usage()
	}
	if err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: png2ico in.{svg,png} out.{ico,png}")
	os.Exit(2)
}

func writePNG(in, out string, size int) error {
	img, err := decodeAt(in, size)
	if err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeICO(in, out string) error {
	img, err := decodeAt(in, 32)
	if err != nil {
		return err
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return err
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
	return os.WriteFile(out, ico.Bytes(), 0o644)
}

func decode(path string) (image.Image, error) {
	return decodeAt(path, 32)
}

func decodeAt(path string, size int) (image.Image, error) {
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return decodeSVG(path, size)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	if b.Dx() == size && b.Dy() == size {
		return img, nil
	}
	return resizeNN(img, size, size), nil
}

func decodeSVG(path string, size int) (image.Image, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// oksvg treats width/height="100%" as a zero viewBox and ignores stroke in style=.
	src := strings.NewReplacer(`width="100%"`, "", `height="100%"`, "").Replace(string(b))
	src = liftStyle(src)
	icon, err := oksvg.ReadIconStream(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	s := float64(size)
	// oksvg scales path coords in SetTarget but not stroke-width.
	if icon.ViewBox.W > 0 {
		scale := s / icon.ViewBox.W
		for i := range icon.SVGPaths {
			icon.SVGPaths[i].LineWidth *= scale
		}
	}
	icon.SetTarget(0, 0, s, s)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scan := rasterx.NewScannerGV(size, size, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(size, size, scan), 1)
	if !hasInk(img) {
		return nil, fmt.Errorf("%s: raster is empty", path)
	}
	return img, nil
}

var (
	styleAttr = regexp.MustCompile(`style="([^"]*)"`)
	rgbColor  = regexp.MustCompile(`^rgb\((\d+),\s*(\d+),\s*(\d+)\)$`)
)

func liftStyle(src string) string {
	return styleAttr.ReplaceAllStringFunc(src, func(m string) string {
		inner := styleAttr.FindStringSubmatch(m)[1]
		var attrs []string
		for _, part := range strings.Split(inner, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			k, v, ok := strings.Cut(part, ":")
			if !ok {
				continue
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			v = strings.TrimSuffix(v, "px")
			if m := rgbColor.FindStringSubmatch(v); m != nil {
				r, _ := strconv.Atoi(m[1])
				g, _ := strconv.Atoi(m[2])
				b, _ := strconv.Atoi(m[3])
				v = fmt.Sprintf("#%02x%02x%02x", r, g, b)
			}
			switch k {
			case "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "fill-rule":
				attrs = append(attrs, k+`="`+v+`"`)
			}
		}
		return strings.Join(attrs, " ")
	})
}

func hasInk(img *image.RGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			return true
		}
	}
	return false
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
