// Command panesoverlay shows the panes overlay look on macOS 26: sidebar, inspector and bottom
// panel float in glass over a gfx canvas scene, with a native toolbar over its top. The toolbar
// covers every zone: sidebar leading and trailing, content leading, center and trailing, and
// inspector leading and trailing. The trailing content button switches the panes between the
// floating look (rounded, inset) and the flush one (glass to the window edges, square). Run:
//
//	go run ./examples/panesoverlay
//
// Drag the scene under the panes and the toolbar to see the picture through the glass; the tint
// on the panes is look.SetGlassTint.
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	viewpkg "github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/look"
)

// markerPad keeps the coloured markers off the glass edges; the rounded corners of the
// floating panes make it required there, flush panes keep it as a plain inset.
const markerPad = 10

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("panes overlay")

		var side, content, insp, bottom *gowt.Panel
		w.Add(
			gowt.VBox(gowt.Ref(&side),
				gowt.Label("Sidebar"),
				gowt.Button("Open"),
				gowt.Button("Share"),
				gowt.VBox(gowt.Padding(markerPad), gowt.Canvas(fill(gowt.RGB{R: 200, G: 60, B: 60}), gowt.Grow()))),
			gowt.VBox(gowt.Ref(&content),
				viewpkg.New(scene(), func(v *viewpkg.View) {}, gowt.Grow())),
			gowt.VBox(gowt.Ref(&insp),
				gowt.Label("Inspector"),
				gowt.Button("Apply"),
				gowt.Button("Reset"),
				gowt.VBox(gowt.Padding(markerPad), gowt.Canvas(fill(gowt.RGB{R: 60, G: 90, B: 220}), gowt.Grow()))),
			gowt.VBox(gowt.Ref(&bottom),
				gowt.Label("Bottom panel"),
				gowt.HBox(gowt.Button("Run"), gowt.Button("Stop")),
				gowt.VBox(gowt.Padding(markerPad), gowt.Canvas(fill(gowt.RGB{R: 220, G: 60, B: 200}), gowt.Grow()))))

		look.SetSidebar(w, side, content, 240)
		look.SetInspector(w, content, insp, 280)
		look.SetBottomPanel(w, content, bottom, 180)

		look.SetPanesOverlay(w, true)

		flush := true // start in the flush look: panes edge to edge, square
		setPanes := func() {
			look.SetSidebarFullHeight(w, flush)
			look.SetInspectorFullHeight(w, flush)
			look.SetBottomPanelFloating(w, !flush)
		}

		toggleSidebar := func() { look.SetSidebarVisible(w, !look.SidebarVisible(w)) }
		toggleInspector := func() { look.SetInspectorVisible(w, !look.InspectorVisible(w)) }
		look.SetToolbar(w,
			look.ToolbarItem{Title: "Add or remove", Sidebar: true, Small: true, Group: []look.ToolbarItem{
				{Title: "Add", Symbol: "plus", Small: true},
				{Title: "Remove", Symbol: "minus", Small: true},
			}},
			look.ToolbarItem{Title: "Items", Symbol: "list.bullet", Sidebar: true, Small: true, ID: "sidebar-items"},
			look.ToolbarItem{Title: "Hide Sidebar", Symbol: "sidebar.left", SidebarEnd: true, Small: true, ID: "hide-sidebar", OnClick: toggleSidebar},
			look.ToolbarItem{Title: "Back", Symbol: "chevron.backward", Nav: true, Small: true, ID: "nav-back"},
			look.ToolbarItem{Title: "Zoom Out", Symbol: "minus.magnifyingglass", Center: true, Small: true, ID: "zoom-out"},
			look.ToolbarItem{Title: "Zoom In", Symbol: "plus.magnifyingglass", Center: true, Small: true, ID: "zoom-in"},
			look.ToolbarItem{Title: "Fit to Window", Symbol: "arrow.up.left.and.arrow.down.right", Small: true, ID: "fit"},
			look.ToolbarItem{Title: "Float/flush panes", Symbol: "rectangle.split.2x1", Small: true, ID: "float-flush",
				OnClick: func() { flush = !flush; setPanes() }},
			look.ToolbarItem{Title: "Hide Code", Symbol: "rectangle.bottomthird.inset.filled", Inspector: true, Small: true, ID: "hide-code"},
			look.ToolbarItem{Space: true, Inspector: true},
			look.ToolbarItem{Title: "Hide Inspector", Symbol: "sidebar.right", Inspector: true, Small: true, ID: "hide-inspector", OnClick: toggleInspector})
		setPanes()

		milk := gowt.RGBA{R: 245, G: 245, B: 250, A: 140}
		look.SetGlassTint(side, milk)
		look.SetGlassTint(insp, milk)
		look.SetGlassTint(bottom, milk)

		w.SetSize(1100, 720)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// scene fills a large area with coloured shapes, so the picture stays visible under the panes
// and the toolbar at any pan and zoom.
func scene() *gfx.Scene {
	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		rec := gfx.NewRecorder()
		rec.FillRect(gfx.R(0, 0, 6000, 4000), gfx.RGB(30, 200, 60)) // green field
		rec.FillRect(gfx.R(200, 150, 1400, 700), gfx.RGB(240, 60, 60))
		rec.FillRect(gfx.R(1800, 200, 1100, 900), gfx.RGB(60, 110, 240))
		rec.FillRect(gfx.R(3400, 150, 1300, 800), gfx.RGB(240, 200, 40))
		rec.FillRect(gfx.R(800, 1200, 1600, 1000), gfx.RGB(40, 210, 210))
		rec.FillRect(gfx.R(3000, 1300, 1400, 1100), gfx.RGB(220, 60, 220))
		rec.FillRect(gfx.R(1300, 2700, 1800, 900), gfx.RGB(250, 140, 30))
		rec.FillRect(gfx.R(3800, 2800, 1500, 900), gfx.RGB(120, 80, 230))
		n := gfx.NewNode(1)
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})
	return scene
}

// fill paints the whole canvas in one colour, a pane's marker.
func fill(c gowt.RGB) func(*gowt.GC) {
	return func(g *gowt.GC) {
		w, h := g.Size()
		g.SetFill(c)
		g.FillRect(0, 0, w, h)
	}
}
