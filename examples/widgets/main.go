// Command widgets shows the common widgets: text fields, buttons, lists, a table, a tree and ranges.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		status := rx.New("")
		w := app.Window("Widgets",
			gowt.Label("", gowt.GrowX(), gowt.BindText(status)),
			gowt.Tabs(gowt.Grow(),
				gowt.Tab("Input", input(status)),
				gowt.Tab("Lists", lists(status)),
				gowt.Tab("Ranges", ranges(status))))
		w.SetSize(520, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func input(status *rx.State[string]) gowt.Option {
	enable := rx.New(true)
	return gowt.Grid(2, gowt.Grow(),
		gowt.Label("User"), gowt.Text(gowt.GrowX(), gowt.Hint("name"), gowt.OnChange(func(s string) { status.Set("text: " + s) })),
		gowt.Label("Password"), gowt.Text(gowt.Password(), gowt.GrowX()),
		gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Grow()),
		gowt.Group("Options", gowt.Span(2), gowt.GrowX(),
			gowt.HBox(
				gowt.Button("Remember me", gowt.Check(), gowt.Checked(true), gowt.BindEnabled(enable)),
				gowt.Switch("Enable options", gowt.Bind(enable)),
				gowt.Button("Small", gowt.Radio()),
				gowt.Button("Large", gowt.Radio()),
				gowt.Link(`<a href="https://example.com">Terms</a>`, gowt.OnClick(func(href string) { status.Set("link: " + href) })),
			)))
}

func lists(status *rx.State[string]) gowt.Option {
	items := []string{"One", "Two", "Three"}
	return gowt.Grid(2, gowt.Grow(),
		gowt.Combo(items, gowt.ReadOnly(), gowt.Selected(0),
			gowt.OnSelect(func(i int) { status.Set(fmt.Sprint("combo: ", items[i])) })),
		gowt.List([]string{"alpha", "beta", "gamma"},
			gowt.OnSelect(func(i int) { status.Set(fmt.Sprint("list: ", i)) })),
		gowt.Table(gowt.Grow(), gowt.Size(0, 120), gowt.Init(func(t *gowt.TableView) {
			t.Column("File", 140)
			t.Column("Size", 60, gowt.Right())
			t.Row("a.txt", "12").SetData("id-a")
			t.Row("b.txt", "340").SetData("id-b")
			t.OnSelect(func(i int) { status.Set(fmt.Sprint("row data: ", t.RowAt(i).Data())) })
		})),
		gowt.Tree(gowt.Grow(), gowt.Init(func(tree *gowt.TreeView) {
			tree.Column("Name", 140)
			tree.Column("Size", 60, gowt.Right())
			tree.SetSort(nil, gowt.SortNone)
			src := tree.Node("src")
			src.Node("main.go").SetTextAt(1, "2 KB")
			src.Node("util.go").SetTextAt(1, "1 KB")
			src.SetExpanded(true)
			tree.Node("README.md").SetTextAt(1, "5 KB")
			tree.OnSelect(func(n *gowt.Node) { status.Set("node: " + n.Text()) })
			tree.OnCellClick(func(c gowt.TreeClick) {
				if c.Node != nil && c.Button == 1 && c.Count == 1 {
					status.Set(fmt.Sprintf("cell: %s col %d", c.Node.Text(), c.Col))
				}
			})
		})))
}

func ranges(status *rx.State[string]) gowt.Option {
	volume := rx.New(30)
	return gowt.Grid(2, gowt.GrowX(),
		gowt.Progress(100, gowt.GrowX(), gowt.Span(2), gowt.Bind(volume)),
		gowt.Label("Volume"), gowt.Scale(0, 100, gowt.GrowX(), gowt.Bind(volume),
			gowt.OnChange(func(v int) { status.Set(fmt.Sprint("scale: ", v)) })),
		gowt.Label("Count"), gowt.Spinner(0, 10, gowt.Value(3),
			gowt.OnChange(func(v int) { status.Set(fmt.Sprint("spinner: ", v)) })),
		gowt.Label("Date"), gowt.DateTime(
			gowt.OnChange(func(t time.Time) { status.Set("date: " + t.Format("2006-01-02")) })))
}
