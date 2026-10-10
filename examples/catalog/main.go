// Command catalog shows one sample of every facade widget. With a widget name as the argument
// (catalog table) it opens only that sample; without arguments it puts all of them in tabs.
// docs/widget-catalog.md lists the names, and `make docs-shots` runs it once per name.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/haiodo/gowt"
)

var fillBoth = gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, VAlign: gowt.AlignFill, GrowX: true, GrowY: true})

type sample struct {
	name  string
	build func(p *gowt.Panel)
}

var samples = []sample{
	{"label", func(p *gowt.Panel) { p.Label("A static label") }},
	{"button", func(p *gowt.Panel) {
		p.Button("Push", nil)
		p.Button("Check", nil, gowt.Check()).SetChecked(true)
		p.Button("Radio", nil, gowt.Radio())
	}},
	{"text", func(p *gowt.Panel) {
		p.Text(gowt.Border()).SetText("Edit me")
		p.Text(gowt.Password(), gowt.Border()).SetText("secret")
		p.Text(gowt.Multiline(), gowt.Border(), gowt.Cell(gowt.GridCell{Height: 60})).SetText("Several\nlines")
	}},
	{"link", func(p *gowt.Panel) { p.Link(`Read the <a href="https://example.com">terms</a>`, func(string) {}) }},
	{"combo", func(p *gowt.Panel) { p.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly()).Select(1) }},
	{"list", func(p *gowt.Panel) {
		p.List([]string{"alpha", "beta", "gamma", "delta"}).SetSelection(1)
	}},
	{"table", func(p *gowt.Panel) {
		t := p.Table(gowt.Border())
		t.Column("File", 120)
		t.Column("Size", 70, gowt.Right())
		t.Row("main.go", "2 KB")
		t.Row("go.mod", "1 KB")
		t.Row("README.md", "5 KB")
	}},
	{"tree", func(p *gowt.Panel) {
		t := p.Tree(gowt.Border())
		src := t.Node("src")
		src.Node("main.go")
		src.Node("util.go")
		src.SetExpanded(true)
		t.Node("README.md")
	}},
	{"scale", func(p *gowt.Panel) { p.Scale(0, 100, 40) }},
	{"slider", func(p *gowt.Panel) { p.Slider(0, 100, 30) }},
	{"spinner", func(p *gowt.Panel) { p.Spinner(0, 99, 7) }},
	{"progress", func(p *gowt.Panel) {
		p.Progress(100).SetValue(60)
		p.Progress(100, gowt.Smooth()).SetValue(30)
	}},
	{"datetime", func(p *gowt.Panel) {
		d := p.DateTime(gowt.AsCalendar())
		d.SetValue(time.Date(2026, 10, 3, 0, 0, 0, 0, time.Local))
	}},
	{"group", func(p *gowt.Panel) {
		g := p.Group("Options")
		g.SetLayout(gowt.Row{Vertical: true, Margin: 8, Spacing: 4})
		g.Button("First", nil, gowt.Check())
		g.Button("Second", nil, gowt.Check())
	}},
	{"panel", func(p *gowt.Panel) {
		in := p.Panel(gowt.Border())
		in.SetLayout(gowt.Row{Margin: 8, Spacing: 8})
		in.Label("Name")
		in.Text(gowt.Border())
	}},
	{"tabs", func(p *gowt.Panel) {
		t := p.Tabs(fillBoth)
		t.Tab("General").Label("First page")
		t.Tab("Advanced").Label("Second page")
	}},
	{"ctabs", func(p *gowt.Panel) {
		t := p.CTabs(fillBoth, gowt.Closable())
		t.Tab("main.go").Label("package main")
		t.Tab("util.go").Label("package util")
	}},
	{"split", func(p *gowt.Panel) {
		s := p.Split(fillBoth)
		s.List([]string{"inbox", "sent", "drafts"})
		s.Text(gowt.Multiline(), gowt.Border()).SetText("Message text")
		s.SetWeights(1, 2)
	}},
	{"scroll", func(p *gowt.Panel) {
		s := p.Scroll(gowt.Cell(gowt.GridCell{Align: gowt.AlignFill, GrowX: true, Width: 140, Height: 110}))
		c := s.Content()
		c.SetLayout(gowt.Row{Vertical: true, Spacing: 4})
		for i := 1; i <= 20; i++ {
			c.Label(fmt.Sprint("Line ", i))
		}
		s.Fit()
	}},
	{"toolbar", func(p *gowt.Panel) {
		t := p.ToolBar(gowt.Flat())
		t.Item("New", nil)
		t.Item("Open", nil)
		t.Separator()
		t.Item("Save", nil)
	}},
	{"coolbar", func(p *gowt.Panel) {
		c := p.CoolBar()
		t1 := c.ToolBar(gowt.Flat())
		t1.Item("Cut", nil)
		t1.Item("Copy", nil)
		c.Add(t1)
		t2 := c.ToolBar(gowt.Flat())
		t2.Item("Undo", nil)
		c.Add(t2)
	}},
	{"sash", func(p *gowt.Panel) {
		p.SetLayout(gowt.Form{})
		p.Label("left", gowt.Anchor(gowt.FormCell{Left: gowt.Percent(0, 0), Right: gowt.Percent(50, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0)}))
		sash := p.Sash(gowt.Vertical(), gowt.Anchor(gowt.FormCell{Left: gowt.Percent(50, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0), Width: 5}))
		p.Label("right", gowt.Anchor(gowt.FormCell{Left: gowt.Beside(sash, 0), Right: gowt.Percent(100, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0)}))
	}},
	{"canvas", func(p *gowt.Panel) {
		p.Canvas(func(g *gowt.GC) {
			g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
			g.FillOval(10, 10, 80, 80)
			g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
			g.Rect(100, 10, 80, 80)
			g.Text("Canvas", 100, 100)
		}, gowt.Cell(gowt.GridCell{Width: 200, Height: 130}))
	}},
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("gowt catalog")
		w.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
		page := func(p *gowt.Panel, s sample) {
			p.SetLayout(gowt.Grid{Columns: 1, Margin: 10, Spacing: 8})
			s.build(p)
		}
		if len(os.Args) > 1 {
			for _, s := range samples {
				if s.name == os.Args[1] {
					w.SetTitle(s.name)
					page(w.Panel(fillBoth), s)
					break
				}
			}
			if len(w.Unwrap().GetChildren()) == 0 {
				log.Fatalf("unknown widget %q", os.Args[1])
			}
		} else {
			tabs := w.Tabs(fillBoth)
			for _, s := range samples {
				page(tabs.Tab(s.name), s)
			}
		}
		if len(os.Args) == 1 {
			w.SetSize(420, 300)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
