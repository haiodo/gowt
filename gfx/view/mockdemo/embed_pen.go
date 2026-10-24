//go:build darwin && gfxmock && gfxmockpen

package main

import (
	"embed"
	"os"
	"path/filepath"
)

// A self-contained demo build carries .pen files copied into embedded/ by the build script; the
// directory is git-ignored, so design files never enter the repository.
//
//go:embed embedded/*.pen
var penFiles embed.FS

func init() {
	dir, err := os.MkdirTemp("", "gfx-mockdemo-")
	if err != nil {
		return
	}
	entries, _ := penFiles.ReadDir("embedded")
	first := ""
	for _, e := range entries {
		b, err := penFiles.ReadFile("embedded/" + e.Name())
		if err != nil {
			return
		}
		p := filepath.Join(dir, e.Name())
		if os.WriteFile(p, b, 0o600) != nil {
			return
		}
		if first == "" && filepath.Ext(e.Name()) == ".pen" && !isLib(e.Name()) {
			first = p
		}
	}
	if first != "" {
		defaultSrc, defaultN, defaultStay = first, 200, true
	}
}

func isLib(name string) bool { return filepath.Ext(name[:len(name)-len(".pen")]) == ".lib" }
