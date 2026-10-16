// Command hellogowt is the smallest gowt program: a window with a field, a button and a label.
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Hello gowt")
		w.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
		name := w.Text(gowt.GrowX(), gowt.Size(200, 0))
		greet := w.Label("", gowt.GrowX(), gowt.Span(2))
		w.Button("Greet", func() { greet.SetText("Hello, " + name.Text() + "!") })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
