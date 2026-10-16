// Command widgets shows the common widgets: text fields, buttons, lists, a table, a tree and ranges.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		var status *gowt.LabelView
		var in, ls, rg *gowt.Panel
		w := app.Window("Widgets",
			gowt.Label("", gowt.GrowX(), gowt.Ref(&status)),
			gowt.Tabs(gowt.Grow(),
				gowt.Tab("Input", gowt.Ref(&in)),
				gowt.Tab("Lists", gowt.Ref(&ls)),
				gowt.Tab("Ranges", gowt.Ref(&rg)),
			),
		)
		say := func(s string) { status.SetText(s) }
		input(in, say)
		lists(ls, say)
		ranges(rg, say)
		w.SetSize(520, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func input(p *gowt.Panel, say func(string)) {
	var name *gowt.TextView
	var remember *gowt.ButtonView
	var enable *gowt.SwitchView
	p.Add(gowt.Grid(2, gowt.Grow(),
		gowt.Label("User"), gowt.Text(gowt.GrowX(), gowt.Ref(&name)),
		gowt.Label("Password"), gowt.Text(gowt.Password(), gowt.GrowX()),
		gowt.Label("Notes"), gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Grow()),
		gowt.Group("Options", gowt.Span(2), gowt.GrowX(),
			gowt.HBox(
				gowt.Button("Remember me", gowt.Check(), gowt.Ref(&remember)),
				gowt.Switch("Enable options", gowt.OnChange(func(on bool) { remember.SetEnabled(on) }), gowt.Ref(&enable)),
				gowt.Button("Small", gowt.Radio()),
				gowt.Button("Large", gowt.Radio()),
				gowt.Link(`<a href="https://example.com">Terms</a>`, gowt.OnClick(func(href string) { say("link: " + href) })),
			),
		),
	))
	name.SetHint("name")
	name.OnChange(func(s string) { say("text: " + s) })
	remember.SetChecked(true)
	enable.SetChecked(true)
}

func lists(p *gowt.Panel, say func(string)) {
	var c *gowt.ComboView
	var l *gowt.ListView
	var t *gowt.TableView
	var tree *gowt.TreeView
	p.Add(gowt.Grid(2, gowt.Grow(),
		gowt.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly(), gowt.Ref(&c)),
		gowt.List([]string{"alpha", "beta", "gamma"}, gowt.Ref(&l)),
		gowt.Table(gowt.Grow(), gowt.Size(0, 120), gowt.Ref(&t)),
		gowt.Tree(gowt.Grow(), gowt.Ref(&tree)),
	))
	c.Select(0)
	c.OnSelect(func(i int) { say(fmt.Sprint("combo: ", c.Items()[i])) })
	l.OnSelect(func(i int) { say(fmt.Sprint("list: ", i)) })

	t.Column("File", 140)
	t.Column("Size", 60, gowt.Right())
	t.Row("a.txt", "12").SetData("id-a")
	t.Row("b.txt", "340").SetData("id-b")
	t.OnSelect(func(i int) { say(fmt.Sprint("row data: ", t.RowAt(i).Data())) })

	tree.Column("Name", 140)
	tree.Column("Size", 60, gowt.Right())
	tree.SetSort(nil, gowt.SortNone)
	src := tree.Node("src")
	src.Node("main.go").SetTextAt(1, "2 KB")
	src.Node("util.go").SetTextAt(1, "1 KB")
	src.SetExpanded(true)
	tree.Node("README.md").SetTextAt(1, "5 KB")
	tree.OnSelect(func(n *gowt.Node) { say("node: " + n.Text()) })
	tree.OnCellClick(func(c gowt.TreeClick) {
		if c.Node != nil && c.Button == 1 && c.Count == 1 {
			say(fmt.Sprintf("cell: %s col %d", c.Node.Text(), c.Col))
		}
	})
}

func ranges(p *gowt.Panel, say func(string)) {
	var bar *gowt.ProgressView
	var volume *gowt.ScaleView
	var count *gowt.SpinnerView
	var d *gowt.DateTimeView
	p.Add(gowt.Grid(2, gowt.GrowX(),
		gowt.Progress(100, gowt.GrowX(), gowt.Span(2), gowt.Ref(&bar)),
		gowt.Label("Volume"), gowt.Scale(0, 100, gowt.Value(30), gowt.GrowX(), gowt.Ref(&volume)),
		gowt.Label("Count"), gowt.Spinner(0, 10, gowt.Value(3), gowt.Ref(&count)),
		gowt.Label("Date"), gowt.DateTime(gowt.Ref(&d)),
	))
	bar.SetValue(30)
	volume.OnChange(func(v int) { bar.SetValue(v); say(fmt.Sprint("scale: ", v)) })
	count.OnChange(func(v int) { say(fmt.Sprint("spinner: ", v)) })
	d.OnChange(func(t time.Time) { say("date: " + t.Format("2006-01-02")) })
}
