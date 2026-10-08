// Command tree shows a Tree (NSOutlineView) with three root items and their children.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		w := app.Window("Tree")
		w.SetLayout(g.Fill{})
		tree := w.Tree(g.Border())

		children := map[string][]string{
			"Fruits":     {"Apple", "Banana", "Cherry"},
			"Vegetables": {"Carrot", "Potato"},
			"Grains":     {"Rice", "Wheat", "Oats"},
		}
		var roots []*g.Node
		for _, name := range []string{"Fruits", "Vegetables", "Grains"} {
			root := tree.Node(name)
			roots = append(roots, root)
			for _, c := range children[name] {
				root.Node(c)
			}
		}
		roots[0].SetExpanded(true)

		tree.OnSelect(func(n *g.Node) {
			if n != nil {
				fmt.Println("Selected:", n.Text())
			}
		})
		tree.OnExpand(func(n *g.Node) {
			if n != nil {
				fmt.Println("Expanded:", n.Text())
			}
		})

		w.SetSize(260, 300)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
