// Command gfxpng renders a scene to hello.png without a window, on macOS, Linux and Windows.
package main

import (
	"log"
	"os"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
)

func main() {
	rec := gfx.NewRecorder()
	rec.FillRRect(gfx.R(0, 0, 160, 80), 10, gfx.RGB(30, 120, 220))
	rec.Text("Hello", 14, 46, gfx.Font{Size: 16}, gfx.White)
	n := gfx.NewNode(0)

	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})

	png, err := offscreen.PNG(scene, n, 2, gfx.White)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("hello.png", png, 0o644); err != nil {
		log.Fatal(err)
	}
}
