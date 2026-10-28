// Command materialpanel puts a code panel under a diagram canvas as the bottom panel of a sidebar
// window (look.SetBottomPanel): on macOS 26 its glass and the canvas colour run on under the floating
// sidebar, and its StyledText paints no background of its own. The sidebar is a Split of a list and
// an outline with a hairline between them (gowt.Divider). The inspector edits the Server box with
// small Number fields (a label or an icon inside the frame, a unit after the value, a drag on the
// label that scrubs the value) and has the small controls of an inspector: a collapsible section
// with borderless icon buttons, segmented controls and toggles.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/rx"
)

// droplet is Lucide's droplet, the opacity symbol of design tools.
const droplet = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22a7 7 0 0 0 7-7c0-2-1-3.9-3-5.5s-3.5-4-4-6.5c-.5 2.5-2 4.9-4 6.5C6 11.1 5 13 5 15a7 7 0 0 0 7 7z"/></svg>`

// The Server box, which the inspector edits.
var (
	serverX, serverY = rx.New(260), rx.New(40)
	serverW, serverH = rx.New(120), rx.New(50)
	opacity          = rx.New(100.0)
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
		var canvas *gowt.CanvasView
		grey := gowt.RGB{R: 142, G: 142, B: 147} // systemGray, the same in light and dark
		field := func(min, max float64, opts ...gowt.Option) gowt.Option {
			return gowt.Number(min, max, append([]gowt.Option{gowt.Small(), gowt.GrowX()}, opts...)...)
		}
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
				gowt.Canvas(diagram, gowt.Ref(&canvas), gowt.Grow(), gowt.Background(paper))),
			gowt.VBox(gowt.Ref(&codePane), gowt.Padding(gowt.Spacing.SM),
				gowt.StyledText(gowt.NoBackground(), gowt.NoBorder(), gowt.Scrollbars(), gowt.LineNumbers(), gowt.Grow(),
					gowt.Init(func(t *gowt.StyledTextView) { t.SetFont(gowt.Type.Mono); t.SetText(source) }))),
			gowt.VBox(gowt.Ref(&insp), gowt.Padding(gowt.Spacing.MD),
				gowt.Label("Server", gowt.Typeface(gowt.Type.BodyStrong)),
				gowt.Label("2 edges", gowt.Foreground(gowt.SecondaryText)),
				gowt.Grid(2, gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
					field(0, 2000, gowt.Prefix("X"), gowt.Bind(serverX), gowt.ID("x")),
					field(0, 2000, gowt.Prefix("Y"), gowt.Bind(serverY), gowt.ID("y")),
					field(20, 2000, gowt.Prefix("W"), gowt.Suffix("px"), gowt.Bind(serverW), gowt.ID("w")),
					field(20, 2000, gowt.Prefix("H"), gowt.Suffix("px"), gowt.Bind(serverH), gowt.ID("h"))),
				field(0, 100, gowt.PrefixImage(icons.FromSVG(app, []byte(droplet), 12, grey)), gowt.Suffix("%"),
					gowt.Step(0.5), gowt.Bind(opacity), gowt.ID("opacity")),
				gowt.Text(gowt.Prefix("Label"), gowt.Small(), gowt.Border(), gowt.Value("Server"), gowt.GrowX(), gowt.ID("label")),
				inspectorControls(app)))
		gowt.Watch(canvas, rx.Compute(func() [5]float64 {
			return [5]float64{float64(serverX.Get()), float64(serverY.Get()), float64(serverW.Get()), float64(serverH.Get()), opacity.Get()}
		}), func([5]float64) { canvas.Redraw() })
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
		look.SetInspector(w, content, insp, gowt.Scaled(260))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// inspectorControls is a collapsible "Layout" section whose header has an eye toggle and + and -
// buttons, over a Fixed/Hug/Fill choice, an alignment choice, momentary segments and Wrap/Clip toggles.
func inspectorControls(app *gowt.App) gowt.Option {
	sizing := rx.New(1)
	shown := rx.New(true)
	img := func(name string) *gowt.Image { return icons.Get(app, name, icons.Size(13)) }
	icon := func(name string) gowt.Option {
		return gowt.Init(func(b *gowt.ButtonView) { b.SetImage(img(name)) })
	}
	eye := gowt.Init(func(b *gowt.ButtonView) {
		gowt.Watch(b, shown, func(on bool) {
			if on {
				b.SetImage(img("eye"))
			} else {
				b.SetImage(img("eye-off"))
			}
		})
	})
	return gowt.Disclosure("Layout", gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
		gowt.Trailing(
			gowt.Button("", gowt.Toggle(), gowt.Borderless(), gowt.Small(), gowt.Tooltip("Show on canvas"), gowt.Bind(shown), eye),
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
			gowt.Segmented([]gowt.Segment{{Image: img("add"), Tooltip: "Add a point"}, {Image: img("remove"), Tooltip: "Remove the point"}}, gowt.Momentary(), gowt.Small())),
		gowt.HBox(gowt.GrowX(), gowt.Gap(gowt.Spacing.SM),
			gowt.Button("Wrap", gowt.Toggle(), gowt.Small(), gowt.Checked(true), icon("menu")),
			gowt.Button("Clip", gowt.Toggle(), gowt.Small(), icon("cut"))))
}

func diagram(g *gowt.GC) {
	sx, sy, sw, sh := serverX.Get(), serverY.Get(), serverW.Get(), serverH.Get()
	boxes := map[string][4]int{"Client": {40, 40, 120, 50}, "Server": {sx, sy, sw, sh}, "Store": {260, 150, 120, 50}}
	g.SetColor(gowt.RGB{R: 120, G: 120, B: 130})
	g.Line(160, 65, sx, sy+sh/2)
	g.Line(sx+sw/2, sy+sh, 320, 150)
	for name, b := range boxes {
		fill := gowt.RGB{R: 230, G: 238, B: 252}
		if name == "Server" { // the box colour at the opacity, over the white paper
			a := opacity.Get() / 100
			fill = gowt.RGB{R: uint8(255 - a*25), G: uint8(255 - a*17), B: uint8(255 - a*3)}
		}
		g.SetFill(fill)
		g.FillRect(b[0], b[1], b[2], b[3])
		g.Rect(b[0], b[1], b[2], b[3])
		g.Text(name, b[0]+12, b[1]+16)
	}
}
