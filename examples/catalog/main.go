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
	name string
	ui   gowt.Option
}

var samples = []sample{
	{"label", gowt.Label("A static label")},
	{"button", gowt.VBox(gowt.Button("Push"), gowt.Button("Check", gowt.Check(), gowt.Checked(true)), gowt.Button("Radio", gowt.Radio()))},
	{"switch", gowt.VBox(gowt.Switch("Wi-Fi", gowt.Checked(true)), gowt.Switch("Bluetooth"))},
	{"text", gowt.VBox(
		gowt.Text(gowt.Border(), gowt.Value("Edit me")),
		gowt.Text(gowt.Password(), gowt.Border(), gowt.Value("secret")),
		gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Size(0, 60), gowt.Value("Several\nlines")))},
	{"link", gowt.Link(`Read the <a href="https://example.com">terms</a>`, gowt.OnClick(func(string) {}))},
	{"combo", gowt.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly(), gowt.Selected(1))},
	{"list", gowt.List([]string{"alpha", "beta", "gamma", "delta"}, gowt.Selected(1))},
	{"table", gowt.Table(gowt.Border(), gowt.Init(func(t *gowt.TableView) {
		t.Column("File", 120)
		t.Column("Size", 70, gowt.Right())
		t.Row("main.go", "2 KB")
		t.Row("go.mod", "1 KB")
		t.Row("README.md", "5 KB")
	}))},
	{"tree", gowt.Tree(gowt.Border(), gowt.Init(func(t *gowt.TreeView) {
		src := t.Node("src")
		src.Node("main.go")
		src.Node("util.go")
		src.SetExpanded(true)
		t.Node("README.md")
	}))},
	{"scale", gowt.Scale(0, 100, gowt.Value(40))},
	{"slider", gowt.Slider(0, 100, gowt.Value(30))},
	{"spinner", gowt.Spinner(0, 99, gowt.Value(7))},
	{"progress", gowt.VBox(gowt.Progress(100, gowt.Value(60)), gowt.Progress(100, gowt.Smooth(), gowt.Value(30)))},
	{"datetime", gowt.DateTime(gowt.AsCalendar(), gowt.Value(time.Date(2026, 10, 3, 0, 0, 0, 0, time.Local)))},
	{"group", gowt.Group("Options", gowt.Button("First", gowt.Check()), gowt.Button("Second", gowt.Check()))},
	{"panel", gowt.HBox(gowt.Border(), gowt.Label("Name"), gowt.Text(gowt.Border()))},
	{"tabs", gowt.Tabs(gowt.Grow(), gowt.Tab("General", gowt.Label("First page")), gowt.Tab("Advanced", gowt.Label("Second page")))},
	{"ctabs", gowt.CTabs(gowt.Grow(),
		gowt.Tab("main.go", gowt.Closable(), gowt.Label("package main")),
		gowt.Tab("util.go", gowt.Closable(), gowt.Label("package util")))},
	{"split", gowt.Split(gowt.Grow(), gowt.Init(func(s *gowt.SplitView) { s.SetWeights(1, 2) }),
		gowt.List([]string{"inbox", "sent", "drafts"}),
		gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Value("Message text")))},
	{"scroll", gowt.Scroll(lines(20)...)},
	{"toolbar", gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) {
		t.Item("New", nil)
		t.Item("Open", nil)
		t.Separator()
		t.Item("Save", nil)
	}))},
	{"coolbar", gowt.CoolBar(
		gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) {
			t.Item("Cut", nil)
			t.Item("Copy", nil)
		})),
		gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) { t.Item("Undo", nil) })))},
	{"sash", sash()},
	{"canvas", gowt.Canvas(func(g *gowt.GC) {
		g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
		g.FillOval(10, 10, 80, 80)
		g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
		g.Rect(100, 10, 80, 80)
		g.Text("Canvas", 100, 100)
	}, gowt.Size(200, 130))},
}

func lines(n int) []gowt.Option {
	l := []gowt.Option{gowt.GrowX(), gowt.Size(140, 110)}
	for i := 1; i <= n; i++ {
		l = append(l, gowt.Label(fmt.Sprint("Line ", i)))
	}
	return l
}

func sash() gowt.Option {
	var sash *gowt.SashView
	full := func(c gowt.FormCell) gowt.Option {
		c.Top, c.Bottom = gowt.Percent(0, 0), gowt.Percent(100, 0)
		return gowt.Anchor(c)
	}
	return gowt.Form(gowt.Grow(),
		gowt.Label("left", full(gowt.FormCell{Left: gowt.Percent(0, 0), Right: gowt.Percent(50, 0)})),
		gowt.Sash(gowt.Vertical(), gowt.Ref(&sash), full(gowt.FormCell{Left: gowt.Percent(50, 0), Width: 5})),
		gowt.Label("right", full(gowt.FormCell{Left: gowt.Beside(&sash, 0), Right: gowt.Percent(100, 0)})))
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
			w.Add(samples[i].ui)
		} else {
			tabs := []gowt.Option{gowt.Grow()}
			for _, s := range samples {
				tabs = append(tabs, gowt.Tab(s.name, s.ui))
			}
			w.Add(gowt.Tabs(tabs...))
			w.SetSize(420, 300)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
