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
		name := w.Text(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 200}))
		greet := w.Label("", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
		w.Button("Greet", func() { greet.SetText("Hello, " + name.Text() + "!") })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
