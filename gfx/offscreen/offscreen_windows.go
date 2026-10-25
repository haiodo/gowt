//go:build windows

package offscreen

import (
	"sync"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/d2d"
)

var (
	mu     sync.Mutex
	shared *d2d.Renderer
)

// get returns the one renderer of the process: RegisterFonts and New share its font collection.
func get(o Options) (*d2d.Renderer, error) {
	mu.Lock()
	defer mu.Unlock()
	if shared == nil {
		r, err := d2d.NewRenderer(d2d.Options{TextCache: true, ShadowSprite: true, LOD: o.LOD, FontDir: o.FontDir})
		if err != nil {
			return nil, err
		}
		shared = r
	}
	return shared, nil
}

// RegisterFonts makes the font files of dir (or the one file) usable by name in this process and
// returns how many were new; call it before the first text is measured.
func RegisterFonts(dir string) (int, error) {
	r, err := get(Options{})
	if err != nil {
		return 0, err
	}
	return r.RegisterFonts(dir)
}

// New returns the Direct2D renderer on a WARP device: no GPU, window or desktop session is needed.
// The renderer is shared by all callers, so LOD is the setting of the first one.
func New(o Options) (gfx.Renderer, error) {
	r, err := get(o)
	if err != nil {
		return nil, err
	}
	if o.FontDir != "" {
		_, _ = r.RegisterFonts(o.FontDir)
	}
	return r, nil
}
