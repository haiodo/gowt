//go:build linux

package offscreen

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
)

// RegisterFonts registers the font files of dir (or the one file) for this process only and returns
// how many were new; call it before the first text is measured.
func RegisterFonts(dir string) (int, error) { return cairo.RegisterFonts(dir) }

// New returns the cairo and Pango renderer; it fails when those libraries are not installed.
func New(o Options) (gfx.Renderer, error) {
	return cairo.NewRenderer(cairo.Options{LOD: o.LOD, FontDir: o.FontDir})
}
