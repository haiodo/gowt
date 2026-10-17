// Command boundary shows errors in place: a Boundary replaces the widgets inside it with a fallback
// when they fail to build or when their bindings panic after a state change.
package main

import (
	"fmt"
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		app.OnError(func(err error) { log.Print(err) }) // caught errors reach OnError too
		parts := rx.New(4)
		share := rx.Compute(func() string { return fmt.Sprintf("Each part: %d", 120/parts.Get()) })

		w := app.Window("Boundary")
		w.Add(
			gowt.Group("Parts", gowt.GrowX(),
				gowt.Scale(0, 8, gowt.GrowX(), gowt.Bind(parts)), // 0 divides by zero
				gowt.Boundary(gowt.GrowX(),
					gowt.Failed(func(err error, reset func()) gowt.Option {
						return gowt.HBox(gowt.GrowX(),
							gowt.Label("Cannot split: "+err.Error(), gowt.GrowX()),
							gowt.Button("Try again", gowt.OnClick(reset)))
					}),
					gowt.Label("", gowt.GrowX(), gowt.BindText(share)))),
			gowt.Group("Default fallback", gowt.GrowX(),
				gowt.Boundary(gowt.GrowX(),
					gowt.Label("A label", gowt.Multiline())))) // Multiline applies to Text only
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
