// Command widgets2 shows a Tabs holding a Combo pair, a Table, and a Scroll.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		ro := []string{"One", "Two", "Three"}
		editable := []string{"Alpha", "Beta", "Gamma"}
		var labels []g.Option
		for i := 1; i <= 30; i++ {
			labels = append(labels, g.Label(fmt.Sprintf("Label %d", i)))
		}
		w := app.Window("Widgets2",
			g.Tabs(g.Grow(), g.Selected(0),
				g.Tab("Combo",
					g.Combo(ro, g.ReadOnly(), g.Selected(0), g.OnSelect(func(i int) { fmt.Println("Combo (read-only) selected:", ro[i]) })),
					g.Combo(editable, g.OnSelect(func(i int) { fmt.Println("Combo (editable) selected:", editable[i]) }))),
				g.Tab("Table", g.Table(g.Grow(), g.Init(func(t *g.TableView) {
					t.Column("Name", 120)
					t.Column("Value", 80)
					for i := 1; i <= 5; i++ {
						t.Row(fmt.Sprintf("Row %d", i), fmt.Sprintf("%d", i*10))
					}
					t.OnSelect(func(i int) {
						if i >= 0 {
							fmt.Println("Table selected:", t.RowAt(i).Text(0))
						}
					})
				}))),
				g.Tab("Scrolled", g.Scroll(append([]g.Option{g.Border(), g.Grow()}, labels...)...))))

		w.SetSize(420, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
