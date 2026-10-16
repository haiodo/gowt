// Command form: a two-column Grid form (Name/Email + OK) with a File > Quit menu bar.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := g.Run(func(app *g.App) {
		name, email := rx.New(""), rx.New("")
		w := app.Window("Form",
			g.Grid(2, g.GrowX(),
				g.Label("Name:"),
				g.Text(g.Border(), g.GrowX(), g.Size(150, 0), g.Bind(name)),
				g.Label("Email:"),
				g.Text(g.Border(), g.GrowX(), g.Bind(email)),
				g.Button("OK", g.OnClick(func() {
					fmt.Println("Name:", name.Get())
					fmt.Println("Email:", email.Get())
				}), g.Span(2))))

		w.MenuBar().Submenu("File").Item("Quit", w.Close)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
