//go:build darwin

package cg

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestRegisterFonts(t *testing.T) {
	bind()
	if _, err := RegisterFonts(t.TempDir() + "/missing"); err == nil {
		t.Error("a missing directory is not an error")
	}
	if n, err := RegisterFonts(t.TempDir()); err != nil || n != 0 {
		t.Errorf("empty directory: %d, %v", n, err)
	}
	// the fonts are not in the repository: with GFX_FONT_DIR the family must resolve to itself
	dir := os.Getenv("GFX_FONT_DIR")
	if dir == "" {
		t.Skip("GFX_FONT_DIR not set")
	}
	r := NewRenderer(Options{})
	a := r.MeasureText("Hamburgefonstiv", gfx.Font{Name: "IBMPlexSans", Size: 40})
	b := r.MeasureText("Hamburgefonstiv", gfx.Font{Name: "NoSuchFont-Probe", Size: 40})
	if a == b {
		t.Error("IBMPlexSans measures as the fallback font")
	}
}
