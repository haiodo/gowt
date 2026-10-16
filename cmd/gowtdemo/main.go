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
)

func main() {
	if err := g.Run(build); err != nil {
		log.Fatal(err)
	}
}

func build(app *g.App) {
	var (
		status *g.LabelView
		tabs   *g.TabsView
		t1, t2 *g.Panel
		t3, t4 *g.Panel
		t5     *g.Panel
	)
	w := app.Window("gowt demo")
	menus(w)
	w.Add(
		g.Label("ready", g.GrowX(), g.Ref(&status)),
		g.Tabs(g.Grow(), g.Ref(&tabs),
			g.Tab("Basics", g.Ref(&t1)),
			g.Tab("Lists", g.Ref(&t2)),
			g.Tab("Containers", g.Ref(&t3)),
			g.Tab("Graphics", g.Ref(&t4)),
			g.Tab("Dialogs", g.Ref(&t5)),
		),
	)
	basics(t1, status)
	lists(t2)
	containers(t3)
	graphics(app, t4)
	dialogs(w, t5)
	tabs.OnSelect(func(i int) { status.SetText(fmt.Sprint("tab ", i)) })
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

func basics(p *g.Panel, status *g.LabelView) {
	var (
		name    *g.TextView
		bar     *g.ProgressView
		scale   *g.ScaleView
		slider  *g.SliderView
		spinner *g.SpinnerView
		date    *g.DateTimeView
	)
	p.Add(g.Grid(2, g.GrowX(),
		g.Text(g.GrowX(), g.Ref(&name)),
		g.Button("Greet", g.OnClick(func() { status.SetText("Hello, " + name.Text()) })),
		g.Link(`Visit <a href="https://github.com/haiodo/gowt">gowt</a>`, g.OnClick(func(href string) { status.SetText(href) }),
			g.Span(2)),
		g.Group("Range widgets", g.GrowX(), g.Span(2),
			g.Grid(2, g.GrowX(),
				g.Progress(100, g.GrowX(), g.Span(2), g.Ref(&bar)),
				g.Label("Scale"), g.Scale(0, 100, g.Value(30), g.GrowX(), g.Ref(&scale)),
				g.Label("Slider"), g.Slider(0, 110, g.Value(20), g.GrowX(), g.Ref(&slider)),
				g.Label("Spinner"), g.Spinner(-10, 10, g.Value(3), g.Ref(&spinner)),
				g.Label("Date"), g.DateTime(g.DropDown(), g.Ref(&date)),
				g.Label("Time"), g.DateTime(g.AsTime()),
			),
		),
		g.Button("Busy", g.Check(), g.Tooltip("a check box")),
		g.Progress(0, g.Indeterminate(), g.GrowX()),
	))
	scale.OnChange(bar.SetValue)
	slider.OnChange(func(v int) { status.SetText(fmt.Sprint("slider ", v)) })
	spinner.OnChange(func(v int) { status.SetText(fmt.Sprint("spinner ", v)) })
	date.OnChange(func(t time.Time) { status.SetText(t.Format(time.DateOnly)) })
}

func lists(p *g.Panel) {
	var (
		combo *g.ComboView
		list  *g.ListView
		out   *g.LabelView
		tbl   *g.TableView
		tree  *g.TreeView
	)
	p.Add(g.Grid(2, g.Grow(),
		g.Combo([]string{"red", "green", "blue"}, g.ReadOnly(), g.GrowX(), g.Ref(&combo)),
		g.List([]string{"one", "two", "three"}, g.MultiSelect(), g.Grow(), g.Ref(&list)),
		g.Label("", g.GrowX(), g.Span(2), g.Ref(&out)),
		g.Table(g.Check(), g.Grow(), g.Ref(&tbl)),
		g.Tree(g.Grow(), g.Ref(&tree)),
	))
	combo.Select(0)
	combo.OnSelect(func(i int) { out.SetText(fmt.Sprint("combo ", i)) })
	list.OnSelect(func(int) { out.SetText(fmt.Sprint("list ", list.Selected())) })

	tbl.Column("Name", 120)
	tbl.Column("Size", 80, g.Right())
	for i := 1; i <= 5; i++ {
		tbl.Row(fmt.Sprint("file", i), fmt.Sprint(i*1024))
	}
	tbl.OnSelect(func(i int) { out.SetText(fmt.Sprint("row ", i, " ", tbl.RowAt(i).Text(0))) })

	root := tree.Node("project")
	src := root.Node("src")
	src.Node("main.go")
	src.Node("util.go")
	root.Node("README")
	root.SetExpanded(true)
	tree.OnSelect(func(n *g.Node) {
		if n != nil {
			out.SetText("node " + n.Text())
		}
	})
}

func containers(p *g.Panel) {
	var (
		split *g.SplitView
		ct    *g.CTabsView
		pages *g.PagesView
		ok    *g.ButtonView
		lines = []g.Option{g.Border(), g.GrowX(), g.Size(0, 80), g.Gap(2)}
	)
	for i := 1; i <= 12; i++ {
		lines = append(lines, g.Label(fmt.Sprint("scrolled line ", i)))
	}
	p.Add(
		g.Split(g.Grow(), g.Size(0, 120), g.Ref(&split),
			g.List([]string{"a", "b", "c"}),
			g.Text(g.Multiline()),
		),
		g.CTabs(g.Border(), g.Grow(), g.Ref(&ct)),
		g.Pages(g.Border(), g.GrowX(), g.Ref(&pages),
			g.Label("page A"),
			g.Button("page B"),
		),
		g.Button("Flip", g.OnClick(func() { pages.Show(1 - pages.Current()) })),
		g.Form(g.GrowX(),
			g.Button("OK", g.Anchor(g.FormCell{Right: g.Percent(100, 0)}), g.Ref(&ok)),
			g.Text(g.Anchor(g.FormCell{Left: g.Percent(0, 0), Right: g.Beside(&ok, -4), Top: g.Same(&ok, 0)})),
		),
		g.Scroll(lines...),
	)
	split.SetWeights(1, 3)
	for i := 1; i <= 3; i++ {
		ct.Add(g.Tab(fmt.Sprint("Doc ", i), g.Closable(), g.Label(fmt.Sprint("page ", i), g.Grow())))
	}
	ct.OnClose(func(i int) bool { return i != 0 })
}

func graphics(app *g.App, p *g.Panel) {
	img, err := app.ImageFrom(swatch())
	if err != nil {
		log.Fatal(err)
	}
	var (
		tb    *g.ToolBarView
		band  *g.ToolBarView
		cv    *g.CanvasView
		angle int
	)
	p.Add(
		g.ToolBar(g.Flat(), g.Ref(&tb)),
		g.CoolBar(g.GrowX(), g.ToolBar(g.Flat(), g.Ref(&band))),
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
		}, g.Grow(), g.Ref(&cv)),
		g.Button("Move circle", g.OnClick(func() { angle += 20; cv.Redraw() })),
	)
	tb.Item("Open", nil).SetImage(img)
	tb.Separator()
	tb.Item("Bold", nil, g.Check())
	tb.Item("Plain", nil, g.Radio())
	band.Item("One", nil)
	band.Item("Two", nil)
	pop := g.PopupMenu(cv)
	pop.Item("Move", func() { angle += 20; cv.Redraw() })
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

func dialogs(w *g.Window, p *g.Panel) {
	var out *g.LabelView
	p.Add(
		g.Label("", g.Size(500, 0), g.Ref(&out)),
		g.Button("Message", g.OnClick(func() {
			a := w.MessageBox(g.Message{Title: "Hi", Text: "Proceed?", Icon: g.IconQuestion, Buttons: g.ButtonsYesNoCancel})
			out.SetText(fmt.Sprint("answer ", a))
		})),
		g.Button("Open files", g.OnClick(func() {
			out.SetText(fmt.Sprint(w.FileDialog(g.FileDialog{Title: "Open", Multi: true,
				Filters: []g.FileFilter{{Name: "Go", Pattern: "*.go"}, {Name: "All", Pattern: "*"}}})))
		})),
		g.Button("Folder", g.OnClick(func() { d, ok := w.DirDialog("Folder", ""); out.SetText(fmt.Sprint(d, ok)) })),
		g.Button("Color", g.OnClick(func() { c, ok := w.ColorDialog(g.RGB{R: 255}); out.SetText(fmt.Sprint(c, ok)) })),
		g.Button("Font", g.OnClick(func() { f, ok := w.FontDialog(g.Font{}); out.SetText(fmt.Sprint(f, ok)) })),
	)
}
