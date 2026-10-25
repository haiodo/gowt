package cairo_test

import (
	"log"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
)

// The cairo renderer needs the libraries but no display. Most programs use gfx/offscreen or
// gfx/view instead.
func ExampleNewRenderer() {
	r, err := cairo.NewRenderer(cairo.Options{})
	if err != nil {
		log.Fatal(err) // libcairo or libpango is not installed
	}
	rec := gfx.NewRecorderWith(r)
	rec.Text("Hi", 0, 12, gfx.Font{Size: 14}, gfx.Black)
	_, _ = gfx.RenderPicture(r, rec.Finish(), 2, gfx.White)
}
