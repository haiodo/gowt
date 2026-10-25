// Command catalog shows one sample of every facade widget. With a widget name as the argument
// (catalog table) it opens only that sample; without arguments it puts all of them in tabs.
// docs/widget-catalog.md lists the names, and `make docs-shots` runs it once per name.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
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
		gowt.Text(gowt.Border(), gowt.Value("Edit me"), gowt.Size(200, 0)),
		gowt.Text(gowt.Password(), gowt.Border(), gowt.Value("secret"), gowt.Size(200, 0)),
		gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Size(200, 60), gowt.Value("Several\nlines")))},
	{"link", gowt.Link(`Read the <a href="https://example.com">terms</a>`, gowt.OnClick(func(string) {}))},
	{"combo", gowt.Combo([]string{"One", "Two", "Three"}, gowt.ReadOnly(), gowt.Selected(1), gowt.Size(160, 0))},
	{"list", gowt.List([]string{"alpha", "beta", "gamma", "delta"}, gowt.Selected(1), gowt.Size(160, 0))},
	{"table", gowt.Table(gowt.Init(func(t *gowt.TableView) {
		t.Column("File", 160)
		t.Column("Size", 80, gowt.Right())
		t.Row("main.go", "2 KB")
		t.Row("go.mod", "1 KB")
		t.Row("README.md", "5 KB")
	}))},
	{"tree", gowt.Tree(gowt.Size(180, 140), gowt.Init(func(t *gowt.TreeView) {
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
	{"panel", gowt.HBox(gowt.Label("Name"), gowt.Text(gowt.Border(), gowt.Size(180, 0)))},
	{"tabs", gowt.Tabs(gowt.Grow(), gowt.Size(0, 110), gowt.Tab("General", gowt.Label("First page")), gowt.Tab("Advanced", gowt.Label("Second page")))},
	{"ctabs", gowt.CTabs(gowt.Grow(), gowt.Size(0, 110),
		gowt.Tab("main.go", gowt.Closable(), gowt.Label("package main")),
		gowt.Tab("util.go", gowt.Closable(), gowt.Label("package util")))},
	{"split", gowt.Split(gowt.Grow(), gowt.Size(0, 110), gowt.Init(func(s *gowt.SplitView) { s.SetWeights(1, 2) }),
		gowt.List([]string{"inbox", "sent", "drafts"}),
		gowt.Text(gowt.Multiline(), gowt.Border(), gowt.Value("Message text")))},
	{"scroll", gowt.Scroll(lines(20)...)},
	{"toolbar", gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) {
		t.Item("New", noop)
		t.Item("Open", noop)
		t.Separator()
		t.Item("Save", noop)
	}))},
	{"coolbar", gowt.CoolBar(coolbarInit(),
		gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) {
			t.Item("Cut", noop)
			t.Item("Copy", noop)
			t.Item("Paste", noop)
		})),
		gowt.ToolBar(gowt.Flat(), gowt.Init(func(t *gowt.ToolBarView) {
			t.Item("Undo", noop)
			t.Item("Redo", noop)
		})))},
	{"sash", sash()},
	{"canvas", gowt.Canvas(func(g *gowt.GC) {
		g.Text("Canvas", 100, 100) // Text paints over the fill color: draw it before the blue one is set
		g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
		g.FillOval(10, 10, 80, 80)
		g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
		g.Rect(100, 10, 80, 80)
	}, gowt.Size(200, 130))},
}

// sizes is the outer size in points (frame included) of the window of every sample; the same on every OS, so
// the docs images of one sample have one size. A name that is not here gets defaultSize.
var sizes = map[string][2]int{
	"coolbar":  {420, 160}, // five tool items in a row: Ubuntu's are the widest
	"datetime": {320, 260}, // the calendar of GTK is the tallest
	"table":    {320, 210},
	"text":     {320, 220},
	"tree":     {320, 210},
}

var defaultSize = [2]int{320, 200}

// frameSlack is what SetSize needs added so that the visible frame has the size in the table: Windows counts the
// invisible resize border into the window rect, and GTK on GNOME (Adwaita) opens a window 6 points narrower and
// 8 taller than asked. Measured on Windows 11 and Ubuntu 26.04; another theme leaves a few pixels, which
// internal/shot.Finish stretches away.
func frameSlack() (w, h int) {
	switch runtime.GOOS {
	case "windows":
		return 14, 7
	case "linux":
		return 6, -8
	}
	return 0, 0
}

func noop() {}

// coolbarInit locks the bands on macOS: the Cocoa port draws SWT's classic grip bars, which look broken
// next to native controls. Unlocked bands can be dragged.
func coolbarInit() gowt.Option {
	return gowt.Init(func(c *gowt.CoolBarView) { c.Unwrap().SetLocked(runtime.GOOS == "darwin") })
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
	// The panes are tinted so that the sash, which draws nothing by itself, has something to divide.
	pane := gowt.Background(gowt.Adaptive{Light: gowt.RGB{R: 226, G: 234, B: 247}, Dark: gowt.RGB{R: 50, G: 56, B: 70}})
	return gowt.Form(gowt.Grow(), gowt.Size(0, 90),
		gowt.Label("left", pane, full(gowt.FormCell{Left: gowt.Percent(0, 0), Right: gowt.Percent(50, 0)})),
		gowt.Sash(gowt.Vertical(), gowt.Ref(&sash), full(gowt.FormCell{Left: gowt.Percent(50, 0), Width: 5})),
		gowt.Label("right", pane, full(gowt.FormCell{Left: gowt.Beside(&sash, 0), Right: gowt.Percent(100, 0)})))
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
			size, ok := sizes[samples[i].name]
			if !ok {
				size = defaultSize
			}
			w.Add(gowt.VBox(gowt.Grow(), samples[i].ui))
			sw, sh := frameSlack()
			w.SetSize(size[0]+sw, size[1]+sh)
			if os.Getenv("GOWT_SNAP") != "" && os.Getenv("GOWT_SNAP_SIZE") == "" {
				// The capture is exported at 2 px per point at exactly this outer size (see internal/shot.Finish).
				os.Setenv("GOWT_SNAP_SIZE", fmt.Sprintf("%dx%d", size[0], size[1]))
			}
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
