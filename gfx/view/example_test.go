//go:build darwin || linux || windows

package view_test

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
)

// A scene in a window. Scrolling pans and pinching zooms without code; the init function sets what
// the app reacts to.
func ExampleNew() {
	scene := gfx.NewScene()
	rec := gfx.NewRecorder()
	rec.FillRRect(gfx.R(0, 0, 160, 80), 10, gfx.RGB(30, 120, 220))
	card := gfx.NewNode(1)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, card)
		tx.SetPicture(card, rec.Finish())
	})

	err := g.Run(func(app *g.App) {
		w := app.Window("Canvas", view.New(scene, func(v *view.View) {
			v.Compositor().SetCamera(gfx.Camera{Zoom: 1})
			v.OnPointer(func(p *view.Pointer) {
				if p.Kind == view.PointerDown && p.Node != nil {
					fmt.Println("pressed node", p.Node.ID(), "at scene point", p.Scene)
				}
			})
		}, g.Grow()))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Drag moves the nodes with the pointer, lands them on release and puts them back on Escape.
func ExampleView_Drag() {
	var scene *gfx.Scene
	_ = view.New(scene, func(v *view.View) {
		v.OnPointer(func(p *view.Pointer) {
			if p.Kind == view.PointerDown && p.Button == 1 && p.Node != nil {
				v.Drag(p, p.Node)
			}
		})
	})
}
