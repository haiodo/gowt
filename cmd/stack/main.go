// Command stack: a Split divides left (buttons) from right (Pages of Groups);
// clicking a button switches the visible panel.
package main

import (
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		var pages *g.PagesView
		var buttons, groups []g.Option
		for i, n := range []string{"A", "B", "C"} {
			buttons = append(buttons, g.Button(n, g.OnClick(func() { pages.Show(i) })))
			groups = append(groups, g.Group("Panel "+n, g.Label("This is panel "+n)))
		}
		w := app.Window("Stack",
			g.Split(g.Grow(), g.Init(func(s *g.SplitView) { s.SetWeights(1, 3) }),
				g.Fill(buttons...),
				g.Pages(append([]g.Option{g.Ref(&pages)}, groups...)...)))

		w.SetSize(500, 300)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
