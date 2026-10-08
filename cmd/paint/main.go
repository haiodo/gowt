// Command paint draws on a Canvas from a paint callback: the drawRect: -> paint event -> GC path.
package main

import (
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		w := app.Window("Paint")
		w.SetLayout(g.Fill{})
		w.Canvas(func(gc *g.GC) {
			gc.SetFill(g.RGB{R: 0, G: 128, B: 0})
			gc.FillRect(20, 20, 120, 80)
			gc.SetColor(g.RGB{R: 255, G: 0, B: 0})
			gc.SetLineWidth(3)
			gc.Line(20, 130, 280, 130)
			gc.SetColor(g.RGB{R: 0, G: 0, B: 255})
			gc.Oval(160, 20, 120, 80)
			gc.SetColor(g.RGB{R: 0, G: 0, B: 0})
			gc.Text("Hello, GC", 20, 150)
		}, g.Background(g.RGB{R: 255, G: 255, B: 255}))
		w.SetSize(300, 220)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
