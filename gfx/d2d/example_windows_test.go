package d2d_test

import (
	"log"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/d2d"
)

// The Direct2D renderer runs on a WARP device and needs no window or desktop session. Most
// programs use gfx/offscreen or gfx/view instead.
func ExampleNewRenderer() {
	r, err := d2d.NewRenderer(d2d.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	rec := gfx.NewRecorderWith(r)
	rec.Text("Hi", 0, 12, gfx.Font{Size: 14}, gfx.Black)
	_, _ = gfx.RenderPicture(r, rec.Finish(), 2, gfx.White)
}
