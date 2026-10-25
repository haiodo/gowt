package pen_test

import (
	"log"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/pen"
)

// Open a design, list its screens and add one to a scene as a subtree you can move, hit-test or
// render.
func ExampleOpen() {
	doc, err := pen.Open("app.pen", pen.Options{})
	if err != nil {
		log.Fatal(err)
	}
	scene := gfx.NewScene()
	var screen *pen.Screen
	scene.Update(func(tx *gfx.Tx) {
		screen, err = doc.Build(tx, nil, doc.Screens()[0].Index)
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(screen.Name, screen.W, screen.H, doc.Unsupported())
}
