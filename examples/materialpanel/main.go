// Command materialpanel puts a code panel under a diagram canvas as the bottom panel of a sidebar
// window (look.SetBottomPanel): on macOS 26 its glass and the canvas colour run on under the floating
// sidebar, and its StyledText paints no background of its own. The sidebar is a Split of a list and
// an outline with a hairline between them (gowt.Divider). The inspector has the small controls of an
// inspector: a section header with borderless icon buttons, segmented controls and toggle buttons.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/rx"
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
		paper := gowt.Adaptive{Light: gowt.RGB{R: 255, G: 255, B: 255}, Dark: gowt.RGB{R: 30, G: 30, B: 30}}
		var sidebar, content, codePane, insp *gowt.Panel
		w.Add(
			gowt.VBox(gowt.Ref(&sidebar), gowt.Padding(0),
				gowt.Split(gowt.Vertical(), gowt.Divider(), gowt.Grow(),
					gowt.List([]string{"Overview", "Network", "Storage"}, gowt.NoBorder(), gowt.Selected(0)),
					gowt.Tree(gowt.NoBorder(), gowt.Init(func(t *gowt.TreeView) {
						n := t.Node("Server")
						n.Node("Client -> Server")
						n.Node("Server -> Store")
						n.SetExpanded(true)
					})))),
			// the content's background runs on under the floating sidebar
			gowt.VBox(gowt.Ref(&content), gowt.Padding(0), gowt.Background(paper),
				gowt.Canvas(diagram, gowt.Grow(), gowt.Background(paper))),
			gowt.VBox(gowt.Ref(&codePane), gowt.Padding(gowt.Spacing.SM),
				gowt.StyledText(gowt.NoBackground(), gowt.NoBorder(), gowt.Scrollbars(), gowt.LineNumbers(), gowt.Grow(),
					gowt.Init(func(t *gowt.StyledTextView) { t.SetFont(gowt.Type.Mono); t.SetText(source) }))),
			gowt.VBox(gowt.Ref(&insp), gowt.Padding(gowt.Spacing.MD),
				gowt.Label("Server", gowt.Typeface(gowt.Type.BodyStrong)),
				gowt.Label("2 edges", gowt.Foreground(gowt.SecondaryText)),
				inspectorControls(app)))
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", Sidebar: true, OnClick: func() {
				look.SetSidebarVisible(w, !look.SidebarVisible(w))
			}},
			look.ToolbarItem{Space: true},
			look.ToolbarItem{Title: "Show or Hide Code", Symbol: "terminal", OnClick: func() {
				look.SetBottomPanelVisible(w, !look.BottomPanelVisible(w))
			}})
		look.SetSidebar(w, sidebar, content, gowt.Scaled(200))
		look.SetBottomPanel(w, content, codePane, gowt.Scaled(200))
		look.SetInspector(w, content, insp, gowt.Scaled(220))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// inspectorControls is a "Layout" section: a header with a toggle and + and - buttons, a
// Fixed/Hug/Fill choice, an alignment choice of icons, momentary +/- segments, Wrap/Clip toggles.
func inspectorControls(app *gowt.App) gowt.Option {
	sizing := rx.New(1)
	shown := rx.New(true)
	img := func(name string) *gowt.Image { return icons.Get(app, name, icons.Size(13)) }
	icon := func(name string) gowt.Option {
		return gowt.Init(func(b *gowt.ButtonView) { b.SetImage(img(name)) })
	}
	return gowt.VBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
		gowt.HBox(gowt.GrowX(), gowt.Gap(0),
			gowt.Label("Layout", gowt.Typeface(gowt.Type.BodyStrong), gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
			gowt.Spacer(),
			gowt.Button("", gowt.Toggle(), gowt.Borderless(), gowt.Small(), gowt.Tooltip("Show on canvas"), gowt.Bind(shown), icon("search")),
			gowt.Button("", gowt.Borderless(), gowt.Small(), gowt.Tooltip("Add"), icon("add")),
			gowt.Button("", gowt.Borderless(), gowt.Small(), gowt.Tooltip("Remove"), icon("remove"))),
		gowt.Segmented([]gowt.Segment{{Text: "Fixed"}, {Text: "Hug"}, {Text: "Fill"}}, gowt.Small(), gowt.GrowX(), gowt.Bind(sizing)),
		gowt.Segmented([]gowt.Segment{
			{Image: img("back"), Tooltip: "Align left"},
			{Image: img("menu"), Tooltip: "Align center"},
			{Image: img("forward"), Tooltip: "Align right"},
		}, gowt.Small(), gowt.GrowX(), gowt.Selected(0)),
		gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
			gowt.Label("Points", gowt.Place(gowt.AlignStart, gowt.AlignCenter)),
			gowt.Spacer(),
			gowt.Segmented([]gowt.Segment{{Text: "+", Tooltip: "Add a point"}, {Text: "-", Tooltip: "Remove the point"}}, gowt.Momentary(), gowt.Small())),
		gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
			gowt.Button("Wrap", gowt.Toggle(), gowt.Small(), gowt.Checked(true), icon("menu")),
			gowt.Button("Clip", gowt.Toggle(), gowt.Small(), icon("cut"))))
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
