// Command tree shows a Tree (NSOutlineView) with three root items and their children.
package main

import (
	"fmt"
	"log"

	g "github.com/haiodo/gowt"
)

func main() {
	err := g.Run(func(app *g.App) {
		children := map[string][]string{
			"Fruits":     {"Apple", "Banana", "Cherry"},
			"Vegetables": {"Carrot", "Potato"},
			"Grains":     {"Rice", "Wheat", "Oats"},
		}
		w := app.Window("Tree", g.Tree(g.Grow(),
			g.OnSelect(func(n *g.Node) {
				if n != nil {
					fmt.Println("Selected:", n.Text())
				}
			}),
			g.Init(func(t *g.TreeView) {
				for i, name := range []string{"Fruits", "Vegetables", "Grains"} {
					root := t.Node(name)
					for _, c := range children[name] {
						root.Node(c)
					}
					root.SetExpanded(i == 0)
				}
				t.OnExpand(func(n *g.Node) {
					if n != nil {
						fmt.Println("Expanded:", n.Text())
					}
				})
			})))

		w.SetSize(260, 300)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
