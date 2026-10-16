// Command form: a two-column Grid form (Name/Email + OK) with a File > Quit menu bar.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		var name, email *g.TextView
		w := app.Window("Form",
			g.Grid(2, g.GrowX(),
				g.Label("Name:"),
				g.Text(g.Border(), g.GrowX(), g.Size(150, 0), g.Ref(&name)),
				g.Label("Email:"),
				g.Text(g.Border(), g.GrowX(), g.Ref(&email)),
				g.Button("OK", func() {
					fmt.Println("Name:", name.Text())
					fmt.Println("Email:", email.Text())
				}, g.Span(2)),
			),
		)

		w.MenuBar().Submenu("File").Item("Quit", w.Close)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
