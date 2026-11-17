package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const url = "https://registry.npmjs.org/monaco-editor/-/monaco-editor-0.52.2.tgz"

func main() {
	dest := filepath.FromSlash("web/vendor/monaco")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		fatal(err)
	}
	resp, err := http.Get(url)
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fatal(fmt.Errorf("monaco download: %s", resp.Status))
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	prefix := "package/min/vs/"
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fatal(err)
		}
		name := filepath.ToSlash(h.Name)
		if h.Typeflag != tar.TypeReg || !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		out := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			fatal(err)
		}
		f, err := os.Create(out)
		if err != nil {
			fatal(err)
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			fatal(err)
		}
		f.Close()
		n++
		_ = path.Base(rel)
	}
	fmt.Printf("vendored %d monaco files into %s\n", n, dest)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
