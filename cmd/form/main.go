// Command form: a two-column Grid form (Name/Email + OK) with a File > Quit menu bar.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		w := app.Window("Form")
		w.SetLayout(g.Grid{Columns: 2, Margin: 5, Spacing: 5})
		field := g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true, Width: 150})

		w.Label("Name:")
		name := w.Text(g.Border(), field)
		w.Label("Email:")
		email := w.Text(g.Border(), field)
		w.Button("OK", func() {
			fmt.Println("Name:", name.Text())
			fmt.Println("Email:", email.Text())
		}, g.Cell(g.GridCell{SpanX: 2}))

		w.MenuBar().Submenu("File").Item("Quit", w.Close)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
