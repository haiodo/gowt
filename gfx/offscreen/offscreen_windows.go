//go:build windows

package offscreen

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/d2d"
)

// RegisterFonts makes the font files of dir (or the one file) usable by name in this process and
// returns how many were new; call it before the first text is measured.
func RegisterFonts(dir string) (int, error) { return d2d.RegisterFonts(dir) }

// New returns a Direct2D renderer on a WARP device: no GPU, window or desktop session is needed.
func New(o Options) (gfx.Renderer, error) {
	r, err := d2d.NewRenderer(d2d.Options{TextCache: true, ShadowSprite: true, LOD: o.LOD, FontDir: o.FontDir})
	if err != nil {
		return nil, err
	}
	return r, nil
}
