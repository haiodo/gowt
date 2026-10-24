//go:build darwin && gfxmock && gfxmockpen

package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/haiodo/gowt/gfx/cg"
)

// embedded/ (git-ignored) holds .pen files and, optionally, fonts/ with the .otf files and their
// licence. The whole directory is embedded so a build without fonts still compiles.
//
//go:embed embedded
var embedded embed.FS

func init() {
	dir, err := os.MkdirTemp("", "gfx-mockdemo-")
	if err != nil {
		return
	}
	entries, _ := embedded.ReadDir("embedded")
	first := ""
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".pen" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if copyOut("embedded/"+e.Name(), p) != nil {
			return
		}
		if first == "" && !isLib(e.Name()) {
			first = p
		}
	}
	if first != "" {
		defaultSrc, defaultN, defaultStay = first, 200, true
	}
	registerFonts(filepath.Join(dir, "fonts"))
}

// registerFonts unpacks embedded/fonts, licence included, and registers it for this process; it
// has to happen before the first text is measured.
func registerFonts(dst string) {
	files, _ := fs.ReadDir(embedded, "embedded/fonts")
	if len(files) == 0 || os.MkdirAll(dst, 0o700) != nil {
		return
	}
	for _, f := range files {
		if !f.IsDir() && copyOut("embedded/fonts/"+f.Name(), filepath.Join(dst, f.Name())) != nil {
			return
		}
	}
	_, _ = cg.RegisterFonts(dst)
}

func copyOut(src, dst string) error {
	b, err := embedded.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o600)
}

func isLib(name string) bool { return strings.HasSuffix(strings.TrimSuffix(name, ".pen"), ".lib") }
