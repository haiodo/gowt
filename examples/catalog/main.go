// Command catalog shows one sample of every facade widget. With a widget name as the argument
// (catalog table) it opens only that sample; without arguments it puts all of them in tabs.
// docs/widget-catalog.md lists the names, and `make docs-shots` runs it once per name.
package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/haiodo/gowt"
)

type sample struct {
	name  string
	build func(p *gowt.Panel)
}

var samples = []sample{
	{"label", func(p *gowt.Panel) { p.Add(gowt.Label("A static label")) }},
	{"button", func(p *gowt.Panel) {
		var check *gowt.ButtonView
		p.Add(
			gowt.Button("Push", nil),
			gowt.Button("Check", nil, gowt.Check(), gowt.Ref(&check)),
			gowt.Button("Radio", nil, gowt.Radio()),
		)
		check.SetChecked(true)
	}},
	{"switch", func(p *gowt.Panel) {
		var wifi *gowt.SwitchView
		p.Add(
			gowt.Switch("Wi-Fi", nil, gowt.Ref(&wifi)),
			gowt.Switch("Bluetooth", nil),
		)
		wifi.SetChecked(true)
	}},
	{"text", func(p *gowt.Panel) {
		var edit, secret, lines *gowt.TextView
		p.Add(
			gowt.Text(gowt.Border(), gowt.Ref(&edit)),
			gowt.Text(gowt.Password(), gowt.Border(), gowt.Ref(&secret)),
			gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Size(0, 60), gowt.Ref(&lines)),
		)
		edit.SetText("Edit me")
		secret.SetText("secret")
		lines.SetText("Several\nlines")
	}},
	{"link", func(p *gowt.Panel) {
		p.Add(gowt.Link(`Read the <a href="https://example.com">terms</a>`, func(string) {}))
	}},
	{"combo", func(p *gowt.Panel) {
		var c *gowt.ComboView
		p.Add(gowt.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly(), gowt.Ref(&c)))
		c.Select(1)
	}},
	{"list", func(p *gowt.Panel) {
		var l *gowt.ListView
		p.Add(gowt.List([]string{"alpha", "beta", "gamma", "delta"}, gowt.Ref(&l)))
		l.SetSelection(1)
	}},
	{"table", func(p *gowt.Panel) {
		var t *gowt.TableView
		p.Add(gowt.Table(gowt.Border(), gowt.Ref(&t)))
		t.Column("File", 120)
		t.Column("Size", 70, gowt.Right())
		t.Row("main.go", "2 KB")
		t.Row("go.mod", "1 KB")
		t.Row("README.md", "5 KB")
	}},
	{"tree", func(p *gowt.Panel) {
		var t *gowt.TreeView
		p.Add(gowt.Tree(gowt.Border(), gowt.Ref(&t)))
		src := t.Node("src")
		src.Node("main.go")
		src.Node("util.go")
		src.SetExpanded(true)
		t.Node("README.md")
	}},
	{"scale", func(p *gowt.Panel) { p.Add(gowt.Scale(0, 100, 40)) }},
	{"slider", func(p *gowt.Panel) { p.Add(gowt.Slider(0, 100, 30)) }},
	{"spinner", func(p *gowt.Panel) { p.Add(gowt.Spinner(0, 99, 7)) }},
	{"progress", func(p *gowt.Panel) {
		var plain, smooth *gowt.ProgressView
		p.Add(
			gowt.Progress(100, gowt.Ref(&plain)),
			gowt.Progress(100, gowt.Smooth(), gowt.Ref(&smooth)),
		)
		plain.SetValue(60)
		smooth.SetValue(30)
	}},
	{"datetime", func(p *gowt.Panel) {
		var d *gowt.DateTimeView
		p.Add(gowt.DateTime(gowt.AsCalendar(), gowt.Ref(&d)))
		d.SetValue(time.Date(2026, 10, 3, 0, 0, 0, 0, time.Local))
	}},
	{"group", func(p *gowt.Panel) {
		p.Add(gowt.Group("Options",
			gowt.Button("First", nil, gowt.Check()),
			gowt.Button("Second", nil, gowt.Check()),
		))
	}},
	{"panel", func(p *gowt.Panel) {
		p.Add(gowt.HBox(gowt.Border(),
			gowt.Label("Name"),
			gowt.Text(gowt.Border()),
		))
	}},
	{"tabs", func(p *gowt.Panel) {
		p.Add(gowt.Tabs(gowt.Grow(),
			gowt.Tab("General", gowt.Label("First page")),
			gowt.Tab("Advanced", gowt.Label("Second page")),
		))
	}},
	{"ctabs", func(p *gowt.Panel) {
		p.Add(gowt.CTabs(gowt.Grow(),
			gowt.Tab("main.go", gowt.Closable(), gowt.Label("package main")),
			gowt.Tab("util.go", gowt.Closable(), gowt.Label("package util")),
		))
	}},
	{"split", func(p *gowt.Panel) {
		var s *gowt.SplitView
		var msg *gowt.TextView
		p.Add(gowt.Split(gowt.Grow(), gowt.Ref(&s),
			gowt.List([]string{"inbox", "sent", "drafts"}),
			gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Ref(&msg)),
		))
		msg.SetText("Message text")
		s.SetWeights(1, 2)
	}},
	{"scroll", func(p *gowt.Panel) {
		lines := make([]gowt.Option, 0, 21)
		lines = append(lines, gowt.GrowX(), gowt.Size(140, 110))
		for i := 1; i <= 20; i++ {
			lines = append(lines, gowt.Label(fmt.Sprint("Line ", i)))
		}
		p.Add(gowt.Scroll(lines...))
	}},
	{"toolbar", func(p *gowt.Panel) {
		var t *gowt.ToolBarView
		p.Add(gowt.ToolBar(gowt.Flat(), gowt.Ref(&t)))
		t.Item("New", nil)
		t.Item("Open", nil)
		t.Separator()
		t.Item("Save", nil)
	}},
	{"coolbar", func(p *gowt.Panel) {
		var t1, t2 *gowt.ToolBarView
		p.Add(gowt.CoolBar(
			gowt.ToolBar(gowt.Flat(), gowt.Ref(&t1)),
			gowt.ToolBar(gowt.Flat(), gowt.Ref(&t2)),
		))
		t1.Item("Cut", nil)
		t1.Item("Copy", nil)
		t2.Item("Undo", nil)
	}},
	{"sash", func(p *gowt.Panel) {
		var sash *gowt.SashView
		p.Add(gowt.Form(gowt.Grow(),
			gowt.Label("left", gowt.Anchor(gowt.FormCell{Left: gowt.Percent(0, 0), Right: gowt.Percent(50, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0)})),
			gowt.Sash(gowt.Vertical(), gowt.Ref(&sash), gowt.Anchor(gowt.FormCell{Left: gowt.Percent(50, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0), Width: 5})),
			gowt.Label("right", gowt.Anchor(gowt.FormCell{Left: gowt.Beside(&sash, 0), Right: gowt.Percent(100, 0), Top: gowt.Percent(0, 0), Bottom: gowt.Percent(100, 0)})),
		))
	}},
	{"canvas", func(p *gowt.Panel) {
		p.Add(gowt.Canvas(func(g *gowt.GC) {
			g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
			g.FillOval(10, 10, 80, 80)
			g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
			g.Rect(100, 10, 80, 80)
			g.Text("Canvas", 100, 100)
		}, gowt.Size(200, 130)))
	}},
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("gowt catalog")
		if len(os.Args) > 1 {
			i := slices.IndexFunc(samples, func(s sample) bool { return s.name == os.Args[1] })
			if i < 0 {
				log.Fatalf("unknown widget %q", os.Args[1])
			}
			w.SetTitle(samples[i].name)
			var page *gowt.Panel
			w.Add(gowt.VBox(gowt.Grow(), gowt.Ref(&page)))
			samples[i].build(page)
		} else {
			pages := make([]*gowt.Panel, len(samples))
			tabs := []gowt.Option{gowt.Grow()}
			for i, s := range samples {
				tabs = append(tabs, gowt.Tab(s.name, gowt.Ref(&pages[i])))
			}
			w.Add(gowt.Tabs(tabs...))
			for i, s := range samples {
				s.build(pages[i])
			}
			w.SetSize(420, 300)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
