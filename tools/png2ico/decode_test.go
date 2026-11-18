package main

import "testing"

func TestDecodeLogoSVGHasInk(t *testing.T) {
	img, err := decode("../../assets/logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0 {
				n++
			}
		}
	}
	if n < 80 || n > 600 {
		t.Fatalf("logo raster ink out of range: %d pixels", n)
	}
}
