// Command widgets shows the common widgets: text fields, buttons, lists, a table, a tree and ranges.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/haiodo/gowt"
)

var fillX = gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true})
var fillBoth = gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true})

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Widgets")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 8, Spacing: 8})
		status := w.Label("", fillX)
		tabs := w.Tabs(fillBoth)
		input(tabs.Tab("Input"), status)
		lists(tabs.Tab("Lists"), status)
		ranges(tabs.Tab("Ranges"), status)
		w.SetSize(520, 380)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func input(p *gowt.Panel, status *gowt.Label) {
	p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
	p.Label("User")
	name := p.Text(fillX)
	name.SetHint("name")
	p.Label("Password")
	p.Text(gowt.Password(), fillX)
	p.Label("Notes")
	p.Text(gowt.Multiline(), gowt.Border(), fillBoth)

	name.OnChange(func(s string) { status.SetText("text: " + s) })

	opts := p.Group("Options", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
	opts.SetLayout(gowt.Row{Margin: 8, Spacing: 12})
	opts.Button("Remember me", nil, gowt.Check()).SetChecked(true)
	opts.Button("Small", nil, gowt.Radio())
	opts.Button("Large", nil, gowt.Radio())
	opts.Link(`<a href="https://example.com">Terms</a>`, func(href string) { status.SetText("link: " + href) })
}

func lists(p *gowt.Panel, status *gowt.Label) {
	p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
	c := p.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly())
	c.Select(0)
	c.OnSelect(func(i int) { status.SetText(fmt.Sprint("combo: ", c.Items()[i])) })
	p.List([]string{"alpha", "beta", "gamma"}).OnSelect(func(i int) { status.SetText(fmt.Sprint("list: ", i)) })

	t := p.Table(gowt.Cell(gowt.GridCell{SpanX: 1, Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true, Height: 120}))
	t.Column("File", 140)
	t.Column("Size", 60, gowt.Right())
	t.Row("a.txt", "12").SetData("id-a")
	t.Row("b.txt", "340").SetData("id-b")
	t.OnSelect(func(i int) { status.SetText(fmt.Sprint("row data: ", t.RowAt(i).Data())) })

	tree := p.Tree(fillBoth)
	src := tree.Node("src")
	src.Node("main.go")
	src.Node("util.go")
	src.SetExpanded(true)
	tree.Node("README.md")
	tree.OnSelect(func(n *gowt.Node) { status.SetText("node: " + n.Text()) })
}

func ranges(p *gowt.Panel, status *gowt.Label) {
	p.SetLayout(gowt.Grid{Columns: 2, Margin: 10, Spacing: 8})
	bar := p.Progress(100, gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
	bar.SetValue(30)
	p.Label("Volume")
	p.Scale(0, 100, 30, fillX).OnChange(func(v int) { bar.SetValue(v); status.SetText(fmt.Sprint("scale: ", v)) })
	p.Label("Count")
	p.Spinner(0, 10, 3).OnChange(func(v int) { status.SetText(fmt.Sprint("spinner: ", v)) })
	p.Label("Date")
	d := p.DateTime()
	d.OnChange(func(t time.Time) { status.SetText("date: " + t.Format("2006-01-02")) })
}
