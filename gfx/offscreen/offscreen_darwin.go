//go:build darwin

package offscreen

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
)

// New returns the CoreGraphics renderer.
func New(o Options) (gfx.Renderer, error) {
	return cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: o.LOD, FontDir: o.FontDir}), nil
}
