//go:build darwin

package offscreen

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
)

// RegisterFonts registers the font files of dir (or the one file) for this process only and returns
// how many were new; call it before the first text is measured.
func RegisterFonts(dir string) (int, error) { return cg.RegisterFonts(dir) }

// New returns the CoreGraphics renderer.
func New(o Options) (gfx.Renderer, error) {
	return cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: o.LOD, FontDir: o.FontDir}), nil
}
