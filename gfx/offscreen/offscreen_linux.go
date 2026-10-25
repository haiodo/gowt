//go:build linux

package offscreen

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
)

// New returns the cairo and Pango renderer; it fails when those libraries are not installed.
func New(o Options) (gfx.Renderer, error) {
	return cairo.NewRenderer(cairo.Options{LOD: o.LOD, FontDir: o.FontDir})
}
