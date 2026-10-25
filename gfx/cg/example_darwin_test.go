package cg_test

import (
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
)

// The CoreGraphics renderer needs no window. Most programs use gfx/offscreen or gfx/view instead.
func ExampleNewRenderer() {
	r := cg.NewRenderer(cg.Options{})
	rec := gfx.NewRecorderWith(r)
	rec.Text("Hi", 0, 12, gfx.Font{Size: 14}, gfx.Black)
	_, _ = gfx.RenderPicture(r, rec.Finish(), 2, gfx.White)
}
