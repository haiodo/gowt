// Command gowtdemo shows the gowt facade widgets, one tab per widget family.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

func main() {
	if err := g.Run(build); err != nil {
		log.Fatal(err)
	}
}

func build(app *g.App) {
	status := rx.New("ready")
	w := app.Window("gowt demo")
	menus(w)
	w.Add(
		g.Label("", g.GrowX(), g.BindText(status)),
		g.Tabs(g.Grow(), g.OnSelect(func(i int) { status.Set(fmt.Sprint("tab ", i)) }),
			g.Tab("Basics", basics(status)),
			g.Tab("Lists", lists()),
			g.Tab("Containers", containers()),
			g.Tab("Graphics", graphics(app)),
			g.Tab("Dialogs", dialogs(w))))
	w.SetSize(760, 560)
	w.Show()
}

func menus(w *g.Window) {
	bar := w.MenuBar()
	file := bar.Submenu("&File")
	file.Item("&Quit", w.Close)
	view := bar.Submenu("&View")
	view.Item("&Grid lines", func() {}, g.Check()).SetChecked(true)
	view.Separator()
	view.Submenu("Zoom").Item("100%", nil, g.Radio())
}

func basics(status *rx.State[string]) g.Option {
	pos := rx.New(30)
	name := rx.New("")
	return g.Grid(2, g.GrowX(),
		g.Text(g.GrowX(), g.Bind(name)),
		g.Button("Greet", g.OnClick(func() { status.Set("Hello, " + name.Peek()) })),
		g.Link(`Visit <a href="https://github.com/haiodo/gowt">gowt</a>`, g.OnClick(status.Set),
			g.Span(2)),
		g.Group("Range widgets", g.GrowX(), g.Span(2),
			g.Grid(2, g.GrowX(),
				g.Progress(100, g.GrowX(), g.Span(2), g.Bind(pos)),
				g.Label("Scale"), g.Scale(0, 100, g.GrowX(), g.Bind(pos)),
				g.Label("Slider"), g.Slider(0, 110, g.Value(20), g.GrowX(), g.OnChange(func(v int) { status.Set(fmt.Sprint("slider ", v)) })),
				g.Label("Spinner"), g.Spinner(-10, 10, g.Value(3), g.OnChange(func(v int) { status.Set(fmt.Sprint("spinner ", v)) })),
				g.Label("Date"), g.DateTime(g.DropDown(), g.OnChange(func(t time.Time) { status.Set(t.Format(time.DateOnly)) })),
				g.Label("Time"), g.DateTime(g.AsTime()))),
		g.Button("Busy", g.Check(), g.Tooltip("a check box")),
		g.Progress(0, g.Indeterminate(), g.GrowX()))
}

func lists() g.Option {
	out := rx.New("")
	return g.Grid(2, g.Grow(),
		g.Combo([]string{"red", "green", "blue"}, g.ReadOnly(), g.GrowX(), g.Selected(0),
			g.OnSelect(func(i int) { out.Set(fmt.Sprint("combo ", i)) })),
		g.List([]string{"one", "two", "three"}, g.MultiSelect(), g.Grow(),
			g.Init(func(l *g.ListView) { l.OnSelect(func(int) { out.Set(fmt.Sprint("list ", l.Selected())) }) })),
		g.Label("", g.GrowX(), g.Span(2), g.BindText(out)),
		g.Table(g.Check(), g.Grow(), g.Init(func(t *g.TableView) {
			t.Column("Name", 120)
			t.Column("Size", 80, g.Right())
			for i := 1; i <= 5; i++ {
				t.Row(fmt.Sprint("file", i), fmt.Sprint(i*1024))
			}
			t.OnSelect(func(i int) { out.Set(fmt.Sprint("row ", i, " ", t.RowAt(i).TextAt(0))) })
		})),
		g.Tree(g.Grow(),
			g.OnSelect(func(n *g.Node) {
				if n != nil {
					out.Set("node " + n.Text())
				}
			}),
			g.Init(func(t *g.TreeView) {
				root := t.Node("project")
				src := root.Node("src")
				src.Node("main.go")
				src.Node("util.go")
				root.Node("README")
				root.SetExpanded(true)
			})))
}

func containers() g.Option {
	var (
		pages *g.PagesView
		ok    *g.ButtonView
		lines = []g.Option{g.Border(), g.GrowX(), g.Size(0, 80), g.Gap(2)}
	)
	for i := 1; i <= 12; i++ {
		lines = append(lines, g.Label(fmt.Sprint("scrolled line ", i)))
	}
	return g.VBox(g.Grow(),
		g.Split(g.Grow(), g.Size(0, 120), g.Init(func(s *g.SplitView) { s.SetWeights(1, 3) }),
			g.List([]string{"a", "b", "c"}),
			g.Text(g.Multiline())),
		g.CTabs(g.Border(), g.Grow(), g.Init(func(c *g.CTabsView) { c.OnClose(func(i int) bool { return i != 0 }) }),
			g.Tab("Doc 1", g.Closable(), g.Label("page 1", g.Grow())),
			g.Tab("Doc 2", g.Closable(), g.Label("page 2", g.Grow())),
			g.Tab("Doc 3", g.Closable(), g.Label("page 3", g.Grow()))),
		g.Pages(g.Border(), g.GrowX(), g.Ref(&pages),
			g.Label("page A"),
			g.Button("page B")),
		g.Button("Flip", g.OnClick(func() { pages.Show(1 - pages.Current()) })),
		g.Form(g.GrowX(),
			g.Button("OK", g.Anchor(g.FormCell{Right: g.Percent(100, 0)}), g.Ref(&ok)),
			g.Text(g.Anchor(g.FormCell{Left: g.Percent(0, 0), Right: g.Beside(&ok, -4), Top: g.Same(&ok, 0)}))),
		g.Scroll(lines...))
}

func graphics(app *g.App) g.Option {
	img, err := app.ImageFrom(swatch())
	if err != nil {
		log.Fatal(err)
	}
	var cv *g.CanvasView
	angle := 0
	move := func() { angle += 20; cv.Redraw() }
	return g.VBox(g.Grow(),
		g.ToolBar(g.Flat(), g.Init(func(tb *g.ToolBarView) {
			tb.Item("Open", nil).SetImage(img)
			tb.Separator()
			tb.Item("Bold", nil, g.Check())
			tb.Item("Plain", nil, g.Radio())
		})),
		g.CoolBar(g.GrowX(), g.ToolBar(g.Flat(), g.Init(func(band *g.ToolBarView) {
			band.Item("One", nil)
			band.Item("Two", nil)
		}))),
		g.Canvas(func(gc *g.GC) {
			w, h := cv.Size()
			gc.SetFill(g.RGB{R: 250, G: 250, B: 240})
			gc.FillRect(0, 0, w, h)
			gc.SetFill(g.RGB{R: 70, G: 130, B: 200})
			gc.FillOval(10+angle, 10, 80, 80)
			gc.SetColor(g.RGB{R: 200, G: 40, B: 40})
			gc.SetLineWidth(3)
			gc.Line(0, 0, w, h)
			gc.Rect(5, 5, w-10, h-10)
			gc.SetColor(g.RGB{})
			gc.Text("Canvas "+fmt.Sprint(angle), 100, 20)
			gc.Image(img, 100, 50)
		}, g.Grow(), g.Ref(&cv), g.Init(func(c *g.CanvasView) { g.PopupMenu(c).Item("Move", move) })),
		g.Button("Move circle", g.OnClick(move)))
}

// swatch is a 32x32 PNG made in memory so the demo needs no files.
func swatch() *bytes.Reader {
	im := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			im.Set(x, y, color.RGBA{uint8(x * 8), uint8(y * 8), 160, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		log.Fatal(err)
	}
	return bytes.NewReader(b.Bytes())
}

func dialogs(w *g.Window) g.Option {
	out := rx.New("")
	return g.VBox(
		g.Label("", g.Size(500, 0), g.BindText(out)),
		g.Button("Message", g.OnClick(func() {
			a := w.MessageBox(g.Message{Title: "Hi", Text: "Proceed?", Icon: g.IconQuestion, Buttons: g.ButtonsYesNoCancel})
			out.Set(fmt.Sprint("answer ", a))
		})),
		g.Button("Open files", g.OnClick(func() {
			out.Set(fmt.Sprint(w.FileDialog(g.FileDialog{Title: "Open", Multi: true,
				Filters: []g.FileFilter{{Name: "Go", Pattern: "*.go"}, {Name: "All", Pattern: "*"}}})))
		})),
		g.Button("Folder", g.OnClick(func() { d, ok := w.DirDialog("Folder", ""); out.Set(fmt.Sprint(d, ok)) })),
		g.Button("Color", g.OnClick(func() { c, ok := w.ColorDialog(g.RGB{R: 255}); out.Set(fmt.Sprint(c, ok)) })),
		g.Button("Font", g.OnClick(func() { f, ok := w.FontDialog(g.Font{}); out.Set(fmt.Sprint(f, ok)) })))
}
