package offscreen_test

import (
	"log"
	"os"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
)

// One call from a scene to the bytes of a PNG, on any OS, without a window.
func ExamplePNG() {
	rec := gfx.NewRecorder()
	rec.FillRRect(gfx.R(0, 0, 160, 80), 10, gfx.RGB(30, 120, 220))
	rec.Text("Hello", 14, 46, gfx.Font{Size: 16}, gfx.White)
	n := gfx.NewNode(0)
	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})

	png, err := offscreen.PNG(scene, n, 2, gfx.White) // nil instead of n: the whole scene
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("hello.png", png, 0o644); err != nil {
		log.Fatal(err)
	}
}

// New is for a fixed area, a Raster, or font registration; Default shares one renderer.
func ExampleNew() {
	r, err := offscreen.New(offscreen.Options{FontDir: "fonts"})
	if err != nil {
		log.Fatal(err) // no cairo and Pango on this Linux, or an OS without a backend
	}
	pic := gfx.NewRecorder()
	pic.Text("Hi", 0, 12, gfx.Font{Size: 14}, gfx.Black)
	raster, err := gfx.RenderPicture(r, pic.Finish(), 1, gfx.White)
	if err != nil {
		log.Fatal(err)
	}
	_ = raster.Image() // an *image.NRGBA
}
