// Command widgets2 shows a Tabs holding a Combo pair, a Table, and a Scroll.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		var readOnly, editable *g.ComboView
		var table *g.TableView
		var tabs *g.TabsView
		var labels []g.Option
		for i := 1; i <= 30; i++ {
			labels = append(labels, g.Label(fmt.Sprintf("Label %d", i)))
		}
		w := app.Window("Widgets2",
			g.Tabs(g.Grow(), g.Ref(&tabs),
				g.Tab("Combo",
					g.Combo([]string{"One", "Two", "Three"}, g.ReadOnly(), g.Ref(&readOnly)),
					g.Combo([]string{"Alpha", "Beta", "Gamma"}, g.Ref(&editable)),
				),
				g.Tab("Table", g.Table(g.Grow(), g.Ref(&table))),
				g.Tab("Scrolled", g.Scroll(append([]g.Option{g.Border(), g.Grow()}, labels...)...)),
			),
		)
		readOnly.Select(0)
		readOnly.OnSelect(func(int) { fmt.Println("Combo (read-only) selected:", readOnly.Text()) })
		editable.OnSelect(func(int) { fmt.Println("Combo (editable) selected:", editable.Text()) })

		table.Column("Name", 120)
		table.Column("Value", 80)
		for i := 1; i <= 5; i++ {
			table.Row(fmt.Sprintf("Row %d", i), fmt.Sprintf("%d", i*10))
		}
		table.OnSelect(func(i int) {
			if i >= 0 {
				fmt.Println("Table selected:", table.RowAt(i).Text(0))
			}
		})

		tabs.Select(0)
		w.SetSize(420, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
