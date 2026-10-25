// Package offscreen renders a gfx scene to an image without a window, a display or a build tag: it
// picks the headless gfx.Renderer of the running OS (CoreGraphics on macOS, cairo and Pango on
// Linux, Direct2D on Windows). On any other OS every call returns an error.
//
// PNG is the whole flow for one image:
//
//	png, err := offscreen.PNG(scene, node, 2, gfx.White)
//
// New gives the renderer itself, for gfx.RenderSubtree with a fixed area, for a Raster, or for
// measuring text.
package offscreen

import (
	"sync"

	"github.com/haiodo/gowt/gfx"
)

// Options are the settings every backend understands.
type Options struct {
	// LOD skips text, shadows and dashes that are unreadable at the scale drawn.
	LOD bool
	// FontDir is a directory (or one file) of fonts to register for this process; the environment
	// variable GFX_FONT_DIR stands in when it is empty.
	FontDir string
}

var (
	defOnce sync.Once
	defR    gfx.Renderer
	defErr  error
)

// Default returns the renderer of the process, made on first use with the zero Options. Use it
// where one renderer is enough; it is safe for concurrent use.
func Default() (gfx.Renderer, error) {
	defOnce.Do(func() { defR, defErr = New(Options{}) })
	return defR, defErr
}

// PNG draws root and its subtree (the whole scene when root is nil) at scale device pixels per
// scene unit on the background bg and returns the PNG file. The image has the size of the subtree
// bounds; the zero Color as bg leaves it transparent. It commits the scene's pending changes first.
func PNG(s *gfx.Scene, root *gfx.Node, scale float64, bg gfx.Color) ([]byte, error) {
	r, err := Default()
	if err != nil {
		return nil, err
	}
	raster, err := gfx.RenderSubtree(r, s, root, gfx.Rect{}, scale, bg)
	if err != nil {
		return nil, err
	}
	return raster.PNG()
}
