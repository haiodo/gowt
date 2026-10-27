// Command materialpanel puts a code panel under a diagram canvas in a Split, on the material of the
// sidebar and inspector (look.SetMaterial), with a StyledText that paints no background of its own.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
)

const source = `node "Client" at (40, 40)
node "Server" at (260, 40)
node "Store" at (260, 150)
edge "Client" -> "Server"
edge "Server" -> "Store"`

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Diagram")
		w.SetSize(1000, 640)
		var sidebar, content, codePane, insp *gowt.Panel
		w.Add(
			gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"Overview", "Network", "Storage"}, gowt.NoBorder(), gowt.Grow(), gowt.Selected(0))),
			gowt.VBox(gowt.Ref(&content), gowt.Padding(0), gowt.Gap(0),
				gowt.Split(gowt.Vertical(), gowt.Grow(), gowt.Init(func(s *gowt.SplitView) { s.SetWeights(3, 2) }),
					gowt.Canvas(diagram, gowt.Background(gowt.Adaptive{Light: gowt.RGB{R: 255, G: 255, B: 255}, Dark: gowt.RGB{R: 30, G: 30, B: 30}})),
					gowt.VBox(gowt.Ref(&codePane), gowt.Padding(gowt.Spacing.SM),
						gowt.StyledText(gowt.NoBackground(), gowt.NoBorder(), gowt.Scrollbars(), gowt.LineNumbers(), gowt.Grow(),
							gowt.Init(func(t *gowt.StyledTextView) { t.SetFont(gowt.Type.Mono); t.SetText(source) }))))),
			gowt.VBox(gowt.Ref(&insp), gowt.Padding(gowt.Spacing.MD),
				gowt.Label("Server", gowt.Typeface(gowt.Type.BodyStrong)),
				gowt.Label("2 edges", gowt.Foreground(gowt.SecondaryText))))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Run", Symbol: "play.fill", OnClick: func() {}})
		look.SetSidebar(w, sidebar, content, gowt.Scaled(200))
		look.SetInspector(w, content, insp, gowt.Scaled(220))
		look.SetMaterial(codePane, true)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func diagram(g *gowt.GC) {
	boxes := map[string][2]int{"Client": {40, 40}, "Server": {260, 40}, "Store": {260, 150}}
	g.SetColor(gowt.RGB{R: 120, G: 120, B: 130})
	g.Line(160, 65, 260, 65)
	g.Line(320, 90, 320, 150)
	for name, p := range boxes {
		g.SetFill(gowt.RGB{R: 230, G: 238, B: 252})
		g.FillRect(p[0], p[1], 120, 50)
		g.Rect(p[0], p[1], 120, 50)
		g.Text(name, p[0]+12, p[1]+16)
	}
}
