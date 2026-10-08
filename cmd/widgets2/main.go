// Command widgets2 shows a Tabs holding a Combo pair, a Table, and a Scroll.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		w := app.Window("Widgets2")
		w.SetLayout(g.Fill{})
		tabs := w.Tabs()

		comboTab := tabs.Tab("Combo")
		comboTab.SetLayout(g.Row{Vertical: true, Margin: 3, Spacing: 3})
		readOnly := comboTab.Combo([]string{"One", "Two", "Three"}, g.ReadOnly(), g.Border())
		readOnly.Select(0)
		readOnly.OnSelect(func(int) { fmt.Println("Combo (read-only) selected:", readOnly.Text()) })
		editable := comboTab.Combo([]string{"Alpha", "Beta", "Gamma"}, g.Border())
		editable.OnSelect(func(int) { fmt.Println("Combo (editable) selected:", editable.Text()) })

		tableTab := tabs.Tab("Table")
		tableTab.SetLayout(g.Fill{})
		table := tableTab.Table()
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

		scrollTab := tabs.Tab("Scrolled")
		scrollTab.SetLayout(g.Fill{})
		scroll := scrollTab.Scroll(g.Border())
		scroll.Content().SetLayout(g.Row{Vertical: true, Margin: 3, Spacing: 3})
		for i := 1; i <= 30; i++ {
			scroll.Content().Label(fmt.Sprintf("Label %d", i))
		}
		scroll.Fit()

		tabs.Select(0)
		w.SetSize(420, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
