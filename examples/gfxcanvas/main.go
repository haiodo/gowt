// Command gfxcanvas puts a gfx scene in a window: two cards you can drag, pan with the trackpad or
// wheel, pinch to zoom. Escape while dragging puts a card back.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
)

// card adds a rounded box with a title to the scene at (x, y).
func card(scene *gfx.Scene, id int, x, y float64, title string, fill gfx.Color) {
	rec := gfx.NewRecorder()
	box := gfx.R(0, 0, 160, 80)
	rec.FillShadowRRect(0, 0, 160, 80, 10, fill, gfx.Shadow{DY: 3, Blur: 8, Color: gfx.Black.WithAlpha(0.25)})
	rec.Text(title, 14, 46, gfx.Font{Size: 16}, gfx.White)
	n := gfx.NewNode(id)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
		tx.SetShape(n, gfx.RoundRectPath(box, 10), gfx.HitFill)
		tx.Move(n, x, y)
	})
}

func main() {
	scene := gfx.NewScene()
	card(scene, 1, 40, 40, "Hello", gfx.RGB(30, 120, 220))
	card(scene, 2, 280, 80, "gfx", gfx.RGB(204, 77, 51))

	err := g.Run(func(app *g.App) {
		w := app.Window("Canvas", view.New(scene, func(v *view.View) {
			// Scrolling pans and pinching zooms without any code here.
			v.OnPointer(func(p *view.Pointer) {
				if p.Kind == view.PointerDown && p.Node != nil {
					fmt.Println("grab node", p.Node.ID())
					v.Drag(p, p.Node)
				}
			})
		}, g.Grow()))
		// Window keys work while the canvas has the focus, like any other widget.
		w.OnKey("Cmd+W", w.Close)
		w.SetSize(800, 600)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
