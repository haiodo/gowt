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

var fill = g.Cell(g.GridCell{Align: g.AlignFill, VAlign: g.AlignFill, GrowX: true, GrowY: true})
var growX = g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true})

func main() {
	if err := g.Run(build); err != nil {
		log.Fatal(err)
	}
}

func build(app *g.App) {
	w := app.Window("gowt demo")
	w.SetLayout(g.Grid{Columns: 1, Margin: 6, Spacing: 6})
	menus(w)
	status := w.Label("ready", growX)
	tabs := w.Tabs(fill)
	basics(tabs.Tab("Basics"), status)
	lists(tabs.Tab("Lists"))
	containers(tabs.Tab("Containers"))
	graphics(app, tabs.Tab("Graphics"))
	dialogs(w, tabs.Tab("Dialogs"))
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

func basics(p *g.Panel, status *g.Label) {
	p.SetLayout(g.Grid{Columns: 2, Margin: 8, Spacing: 8})
	name := p.Text(growX)
	p.Button("Greet", func() { status.SetText("Hello, " + name.Text()) })
	p.Link(`Visit <a href="https://github.com/haiodo/gowt">gowt</a>`, func(href string) { status.SetText(href) },
		g.Cell(g.GridCell{SpanX: 2}))

	grp := p.Group("Range widgets", g.Cell(g.GridCell{SpanX: 2, Align: g.AlignFill, GrowX: true}))
	grp.SetLayout(g.Grid{Columns: 2, Margin: 8, Spacing: 8})
	bar := grp.Progress(100, g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true, SpanX: 2}))
	grp.Label("Scale")
	grp.Scale(0, 100, 30, growX).OnChange(bar.SetValue)
	grp.Label("Slider")
	grp.Slider(0, 110, 20, growX).OnChange(func(v int) { status.SetText(fmt.Sprint("slider ", v)) })
	grp.Label("Spinner")
	grp.Spinner(-10, 10, 3).OnChange(func(v int) { status.SetText(fmt.Sprint("spinner ", v)) })
	grp.Label("Date")
	grp.DateTime(g.DropDown()).OnChange(func(t time.Time) { status.SetText(t.Format(time.DateOnly)) })
	grp.Label("Time")
	grp.DateTime(g.AsTime())
	p.Button("Busy", nil, g.Check(), g.Tooltip("a check box"))
	p.Progress(0, g.Indeterminate(), growX)
}

func lists(p *g.Panel) {
	p.SetLayout(g.Grid{Columns: 2, Margin: 8, Spacing: 8})
	combo := p.Combo([]string{"red", "green", "blue"}, g.ReadOnly(), growX)
	list := p.List([]string{"one", "two", "three"}, g.MultiSelect(), fill)
	out := p.Label("", g.Cell(g.GridCell{SpanX: 2, Align: g.AlignFill, GrowX: true}))
	combo.Select(0)
	combo.OnSelect(func(i int) { out.SetText(fmt.Sprint("combo ", i)) })
	list.OnSelect(func(int) { out.SetText(fmt.Sprint("list ", list.Selected())) })

	tbl := p.Table(g.Check(), fill)
	tbl.Column("Name", 120)
	tbl.Column("Size", 80, g.Right())
	for i := 1; i <= 5; i++ {
		tbl.Row(fmt.Sprint("file", i), fmt.Sprint(i*1024))
	}
	tbl.OnSelect(func(i int) { out.SetText(fmt.Sprint("row ", i, " ", tbl.RowAt(i).Text(0))) })

	tree := p.Tree(fill)
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
	p.SetLayout(g.Grid{Columns: 1, Margin: 8, Spacing: 8})
	split := p.Split(g.Cell(g.GridCell{Align: g.AlignFill, VAlign: g.AlignFill, GrowX: true, GrowY: true, Height: 120}))
	split.List([]string{"a", "b", "c"})
	split.Text(g.Multiline())
	split.SetWeights(1, 3)

	ct := p.CTabs(g.Border(), fill)
	for i := 1; i <= 3; i++ {
		page := ct.Tab(fmt.Sprint("Doc ", i), g.Closable())
		page.SetLayout(g.Fill{})
		page.Label(fmt.Sprint("page ", i))
	}
	ct.OnClose(func(i int) bool { return i != 0 })

	stack := p.Panel(g.Border(), growX)
	stack.SetLayout(g.Stack{Margin: 4})
	pages := []g.Widget{stack.Label("page A"), stack.Button("page B", nil)}
	stack.ShowTop(pages[0])
	flipped := false
	p.Button("Flip", func() {
		flipped = !flipped
		if flipped {
			stack.ShowTop(pages[1])
		} else {
			stack.ShowTop(pages[0])
		}
	})

	form := p.Panel(growX)
	form.SetLayout(g.Form{Margin: 4, Spacing: 4})
	ok := form.Button("OK", nil, g.Anchor(g.FormCell{Right: g.Percent(100, 0)}))
	form.Text(g.Anchor(g.FormCell{Left: g.Percent(0, 0), Right: g.Beside(ok, -4), Top: g.Same(ok, 0)}))

	scroll := p.Scroll(g.Border(), g.Cell(g.GridCell{Align: g.AlignFill, GrowX: true, Height: 80}))
	body := scroll.Content()
	body.SetLayout(g.Row{Vertical: true, Spacing: 2})
	for i := 1; i <= 12; i++ {
		body.Label(fmt.Sprint("scrolled line ", i))
	}
	scroll.Fit()
}

func graphics(app *g.App, p *g.Panel) {
	p.SetLayout(g.Grid{Columns: 1, Margin: 8, Spacing: 8})
	img, err := app.ImageFrom(swatch())
	if err != nil {
		log.Fatal(err)
	}
	tb := p.ToolBar(g.Flat())
	tb.Item("Open", nil).SetImage(img)
	tb.Separator()
	tb.Item("Bold", nil, g.Check())
	tb.Item("Plain", nil, g.Radio())

	cool := p.CoolBar(growX)
	band := cool.ToolBar(g.Flat())
	band.Item("One", nil)
	band.Item("Two", nil)
	cool.Add(band)

	angle := 0
	var cv *g.Canvas
	cv = p.Canvas(func(gc *g.GC) {
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
	}, fill)
	pop := g.PopupMenu(cv)
	pop.Item("Move", func() { angle += 20; cv.Redraw() })
	p.Button("Move circle", func() { angle += 20; cv.Redraw() })
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
	p.SetLayout(g.Row{Vertical: true, Margin: 8, Spacing: 6})
	out := p.Label("", g.InRow(g.RowCell{Width: 500}))
	p.Button("Message", func() {
		a := w.MessageBox(g.Message{Title: "Hi", Text: "Proceed?", Icon: g.IconQuestion, Buttons: g.ButtonsYesNoCancel})
		out.SetText(fmt.Sprint("answer ", a))
	})
	p.Button("Open files", func() {
		out.SetText(fmt.Sprint(w.FileDialog(g.FileDialog{Title: "Open", Multi: true,
			Filters: []g.FileFilter{{Name: "Go", Pattern: "*.go"}, {Name: "All", Pattern: "*"}}})))
	})
	p.Button("Folder", func() { d, ok := w.DirDialog("Folder", ""); out.SetText(fmt.Sprint(d, ok)) })
	p.Button("Color", func() { c, ok := w.ColorDialog(g.RGB{R: 255}); out.SetText(fmt.Sprint(c, ok)) })
	p.Button("Font", func() { f, ok := w.FontDialog(g.Font{}); out.SetText(fmt.Sprint(f, ok)) })
}
