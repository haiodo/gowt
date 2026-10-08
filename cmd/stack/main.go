// Command stack: a Split divides left (buttons) from right (a Stack of Groups);
// clicking a button switches the visible panel.
package main

import (
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		w := app.Window("Stack")
		w.SetLayout(g.Fill{})
		split := w.Split()

		left := split.Panel.Panel()
		left.SetLayout(g.Fill{})
		right := split.Panel.Panel()
		right.SetLayout(g.Stack{})

		var groups []*g.Group
		for _, n := range []string{"A", "B", "C"} {
			grp := right.Group("Panel " + n)
			grp.SetLayout(g.Fill{})
			grp.Label("This is panel " + n)
			groups = append(groups, grp)
		}
		right.ShowTop(groups[0])
		split.SetWeights(1, 3)

		for i, n := range []string{"A", "B", "C"} {
			left.Button(n, func() { right.ShowTop(groups[i]) })
		}

		w.SetSize(500, 300)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
