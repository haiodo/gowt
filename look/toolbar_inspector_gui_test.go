package look_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// The Inspector items sit over the inspector and follow its edge through a drag and a hide; the small
// zoom group and text item before them stay over the content, and the text item follows its state.
func TestToolbarInspectorItems(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	zoom := rx.New("100%")
	clicks := 0
	var errs []string
	fail := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("toolbar inspector")
			var content, insp *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&content), gowt.Label("canvas")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(12), gowt.Label("inspector")))
			look.SetInspector(w, content, insp, 260)
			nop := func() {}
			look.SetToolbar(w,
				look.ToolbarItem{Title: "Share", ID: "share", Symbol: "square.and.arrow.up", OnClick: nop},
				look.ToolbarItem{Small: true, Group: []look.ToolbarItem{
					{Title: "Zoom Out", ID: "out", Symbol: "minus", OnClick: nop},
					{Title: "Zoom In", ID: "in", Symbol: "plus", OnClick: nop}}},
				look.ToolbarItem{Title: "Zoom", ID: "zoom", Text: zoom, Small: true, OnClick: func() { clicks++ }},
				look.ToolbarItem{Title: "Info", ID: "info", Symbol: "info.circle", Inspector: true, OnClick: nop},
				look.ToolbarItem{Space: true, Inspector: true},
				look.ToolbarItem{Title: "Inspector", ID: "toggle", Symbol: "sidebar.right", Inspector: true, OnClick: func() {
					look.SetInspectorVisible(w, !look.InspectorVisible(w))
				}})
			w.SetSize(900, 600)
			w.Show()
			d := app.Unwrap()
			node := func(step, id string) qatree.Node {
				ns, ferr := qatree.Find(d, qatree.Selector{ID: id})
				if ferr != nil || len(ns) != 1 {
					fail("%s: find %s: %v, %d nodes", step, id, ferr, len(ns))
					return qatree.Node{}
				}
				if ns[0].Bounds[2] <= 0 {
					fail("%s: %s has no bounds", step, id)
				}
				return ns[0]
			}
			// check reports where the items are against the inspector edge, the right end when it is hidden.
			check := func(step string) (info qatree.Node) {
				r := w.Unwrap().GetClientArea()
				edge := int(r.X + r.Width)
				if look.InspectorVisible(w) {
					edge = int(insp.Unwrap().GetBounds().X)
				}
				var before int
				for _, id := range []string{"share", "out", "in", "zoom"} {
					n := node(step, id)
					if end := n.Bounds[0] + n.Bounds[2]; end > edge {
						fail("%s: %s ends at %d, past the inspector edge %d", step, id, end, edge)
					}
					before = max(before, n.Bounds[0]+n.Bounds[2])
				}
				info, toggle := node(step, "info"), node(step, "toggle")
				for _, n := range []qatree.Node{info, toggle} {
					if !n.Visible || n.Bounds[0]+n.Bounds[2] > int(r.X+r.Width) {
						fail("%s: %s at %v, visible %v, outside the client area %v", step, n.ID, n.Bounds, n.Visible, r)
					}
					if look.InspectorVisible(w) && n.Bounds[0] < edge {
						fail("%s: %s starts at %d, before the inspector edge %d", step, n.ID, n.Bounds[0], edge)
					}
					if n.Bounds[0] < before {
						fail("%s: %s starts at %d, before the content items end at %d", step, n.ID, n.Bounds[0], before)
					}
				}
				if toggle.Bounds[0] <= info.Bounds[0] {
					fail("%s: the toggle %v is not right of info %v", step, toggle.Bounds, info.Bounds)
				}
				return info
			}
			app.After(500*time.Millisecond, func() {
				start := check("start")
				zoom.Set("125%")
				if e := qatree.Act(d, qatree.Selector{ID: "zoom"}, "click", json.RawMessage(nil)); e != nil {
					fail("click zoom: %v", e)
				}
				sash := inspectorSash(w, insp)
				send := func(typ, x int32) {
					e := swt.NewEvent()
					e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
					sash.NotifyListeners(typ, e)
				}
				x0 := sash.GetBounds().X
				send(swt.MouseDown, x0)
				send(swt.Selection, x0-80)
				send(swt.MouseUp, x0-80)
				app.After(500*time.Millisecond, func() {
					if look.InspectorWidth(w) != 340 {
						fail("drag: inspector width %d, want 340", look.InspectorWidth(w))
					}
					dragged := check("dragged")
					if moved := start.Bounds[0] - dragged.Bounds[0]; moved < 70 || moved > 90 {
						fail("dragged: info moved by %d, want about 80", moved)
					}
					if n := node("dragged", "zoom"); n.Text != "125%" || clicks != 1 {
						fail("text item shows %q after 125%%, clicked %d times", n.Text, clicks)
					}
					// The strip is right-aligned, so with an inspector about as wide as the strip the
					// items stay put: hiding it shows in the content taking the room back.
					draggedWidth := content.Unwrap().GetBounds().Width
					look.SetInspectorVisible(w, false)
					var poll func(tries int)
					poll = func(tries int) {
						cw := content.Unwrap().GetBounds().Width
						if !look.InspectorVisible(w) && cw > draggedWidth || tries == 0 {
							defer w.Close()
							hidden := check("hidden")
							if look.InspectorVisible(w) {
								fail("hidden: the inspector is still visible")
							}
							if cw <= draggedWidth {
								fail("hidden: content %d wide, not wider than the %d it had beside the inspector", cw, draggedWidth)
							}
							if hidden.Bounds[0] < dragged.Bounds[0] {
								fail("hidden: info at %v, left of where it was over the inspector %v", hidden.Bounds, dragged.Bounds)
							}
							return
						}
						app.After(50*time.Millisecond, func() { poll(tries - 1) })
					}
					poll(40)
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Error(e)
	}
}
