package look_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// inspectorWindow opens a 900 by 600 point window with an inspector 260 points wide, with a sidebar
// 220 points wide when sidebar is set, and a toolbar whose last item toggles the inspector; check runs
// once it is on screen.
func inspectorWindow(t *testing.T, sidebar bool, check func(app *gowt.App, w *gowt.Window, side, content, insp *gowt.Panel)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("inspector")
			var side, content, insp *gowt.Panel
			if sidebar {
				w.Add(gowt.VBox(gowt.Ref(&side), gowt.List([]string{"a", "b"}, gowt.Grow())))
			}
			w.Add(gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(12), gowt.Label("inspector")))
			look.SetInspector(w, content, insp, 260)
			if sidebar {
				look.SetSidebar(w, side, content, 220)
			}
			look.SetToolbar(w, look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up"},
				look.ToolbarItem{Space: true},
				look.ToolbarItem{Title: "Inspector", Symbol: "sidebar.right", OnClick: func() {
					look.SetInspectorVisible(w, !look.InspectorVisible(w))
				}})
			w.SetSize(900, 600)
			w.Show()
			app.After(300*time.Millisecond, func() {
				defer w.Close()
				check(app, w, side, content, insp)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// inspectorSash is the sash on the inspector's left edge.
func inspectorSash(w *gowt.Window, insp *gowt.Panel) *swt.Sash {
	x := insp.Unwrap().GetBounds().X
	var best *swt.Sash
	for _, c := range w.Unwrap().GetChildren() {
		if s, ok := c.Impl().(*swt.Sash); ok && (best == nil || abs(s.GetBounds().X-x) < abs(best.GetBounds().X-x)) {
			best = s
		}
	}
	return best
}

func abs(v int32) int32 { return max(v, -v) }

func TestInspectorHide(t *testing.T) {
	inspectorWindow(t, false, func(app *gowt.App, w *gowt.Window, _, content, insp *gowt.Panel) {
		var changes []string
		look.OnInspectorChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		r := w.Unwrap().GetClientArea()
		ib, cb := insp.Unwrap().GetBounds(), content.Unwrap().GetBounds()
		if !look.InspectorVisible(w) || look.InspectorWidth(w) != 260 || ib.Width != 260 || ib.X+ib.Width != r.X+r.Width {
			t.Errorf("start: visible %v, width %d, inspector %v in client area %v", look.InspectorVisible(w), look.InspectorWidth(w), ib, r)
		}
		if cb.X+cb.Width > ib.X || cb.X != r.X {
			t.Errorf("start: content %v overlaps the inspector %v", cb, ib)
		}
		if look.SidebarVisible(w) || look.SidebarWidth(w) != 0 {
			t.Error("an inspector window reports a sidebar")
		}

		look.SetInspectorVisible(w, false)
		app.Flush()
		cb = content.Unwrap().GetBounds()
		if look.InspectorVisible(w) || insp.Unwrap().GetVisible() || cb.X != r.X || cb.Width != r.Width {
			t.Errorf("hidden: visible %v, inspector shown %v, content %v in client area %v",
				look.InspectorVisible(w), insp.Unwrap().GetVisible(), cb, r)
		}
		if inspectorSash(w, insp).GetVisible() {
			t.Error("a hidden inspector keeps its sash")
		}
		look.SetInspectorVisible(w, false) // no change, no callback

		look.SetInspectorVisible(w, true)
		look.SetInspectorWidth(w, 50)
		app.Flush()
		if got := look.InspectorWidth(w); got != 160 || insp.Unwrap().GetBounds().Width != 160 {
			t.Errorf("width 50: got %d, laid out %d, want 160", got, insp.Unwrap().GetBounds().Width)
		}
		if want := "[false 260 true 260 true 160]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
	})
}

// A drag delivered as the events a Sash sends: Selection with the new sash x between the mouse
// down and up.
func TestInspectorDragEvents(t *testing.T) {
	inspectorWindow(t, false, func(app *gowt.App, w *gowt.Window, _, content, insp *gowt.Panel) {
		var changes []string
		look.OnInspectorChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		sash := inspectorSash(w, insp)
		send := func(typ int32, x int32) int32 {
			e := swt.NewEvent()
			e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
			sash.NotifyListeners(typ, e)
			return e.X
		}
		x0 := sash.GetBounds().X
		send(swt.MouseDown, 0)
		if got := send(swt.Selection, x0-80); got != x0-80 || look.InspectorWidth(w) != 340 {
			t.Errorf("drag left by 80: sash at %d, width %d", got, look.InspectorWidth(w))
		}
		if len(changes) != 0 {
			t.Errorf("changes during the drag: %v", changes)
		}
		r := w.Unwrap().GetClientArea()
		send(swt.Selection, r.X+r.Width)
		if !look.InspectorVisible(w) || look.InspectorWidth(w) != 160 {
			t.Errorf("dragged to the right edge: visible %v, width %d; want the drag to stop at 160", look.InspectorVisible(w), look.InspectorWidth(w))
		}
		send(swt.MouseUp, 0)
		if want := "[true 160]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
		send(swt.MouseDown, 0)
		send(swt.Selection, r.X)
		send(swt.MouseUp, 0)
		if cw := content.Unwrap().GetBounds().Width; !look.InspectorVisible(w) || cw < 200 {
			t.Errorf("dragged to the left edge: visible %v, content %d wide", look.InspectorVisible(w), cw)
		}
	})
}

// A real drag of the inspector edge by 60 points to the left, through the system's input (postMouse).
func TestInspectorPostedDrag(t *testing.T) {
	skip := ""
	inspectorWindow(t, false, func(app *gowt.App, w *gowt.Window, _, _, insp *gowt.Panel) {
		if skip = postBlocked(app); skip != "" {
			return
		}
		b := inspectorSash(w, insp).GetBounds()
		p := w.Unwrap().ToDisplay(b.X+b.Width/2, b.Y+b.Height/2)
		d := app.Unwrap()
		post := func(typ int32, x int32) bool {
			ok := postMouse(app, typ, x, p.Y)
			for range 5 {
				for d.ReadAndDispatch() {
				}
				time.Sleep(20 * time.Millisecond)
			}
			return ok
		}
		var changes []string
		look.OnInspectorChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		ok := post(swt.MouseMove, p.X) && post(swt.MouseDown, p.X)
		for i := int32(1); ok && i <= 6; i++ {
			ok = post(swt.MouseMove, p.X-10*i)
		}
		ok = ok && post(swt.MouseUp, p.X-60)
		if !ok {
			skip = "the mouse events could not be posted"
			return
		}
		if got := look.InspectorWidth(w); got != 320 {
			t.Errorf("drag by 60: width %d, want 320", got)
		}
		if want := "[true 320]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
	})
	if skip != "" {
		t.Skip(skip)
	}
}

// Sidebar, content and inspector side by side: each pane hides on its own, a drag of either edge
// leaves the content 200 points, and the window minimum counts both panes.
func TestInspectorWithSidebar(t *testing.T) {
	inspectorWindow(t, true, func(app *gowt.App, w *gowt.Window, side, content, insp *gowt.Panel) {
		s := w.Unwrap()
		r := s.GetClientArea()
		sb, cb, ib := side.Unwrap().GetBounds(), content.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
		if sb.X+sb.Width > cb.X || cb.X+cb.Width > ib.X || ib.X+ib.Width != r.X+r.Width {
			t.Errorf("sidebar %v, content %v, inspector %v in %v", sb, cb, ib, r)
		}
		trim := s.GetSize().X - r.Width
		both := (cb.X - r.X) + 200 + (r.X + r.Width - cb.X - cb.Width) + trim
		if got := s.GetMinimumSize().X; got != both {
			t.Errorf("minimum width %d, want %d", got, both)
		}
		look.SetInspectorVisible(w, false)
		app.Flush()
		cb = content.Unwrap().GetBounds()
		if cb.X+cb.Width != r.X+r.Width || !look.SidebarVisible(w) || cb.X <= sb.X {
			t.Errorf("inspector hidden: content %v", cb)
		}
		look.SetSidebarVisible(w, false)
		app.Flush()
		if cb = content.Unwrap().GetBounds(); cb.X != r.X || cb.Width != r.Width {
			t.Errorf("both hidden: content %v in %v", cb, r)
		}
		if got := s.GetMinimumSize().X; got != 200+trim {
			t.Errorf("both hidden: minimum width %d, want %d", got, 200+trim)
		}
		look.SetSidebarVisible(w, true)
		look.SetInspectorVisible(w, true)
		app.Flush()
		sash := inspectorSash(w, insp)
		e := swt.NewEvent()
		e.Y, e.Button = sash.GetBounds().Y, 1
		sash.NotifyListeners(swt.MouseDown, e)
		e.X = r.X
		sash.NotifyListeners(swt.Selection, e)
		sash.NotifyListeners(swt.MouseUp, e)
		if cw := content.Unwrap().GetBounds().Width; cw != 200 {
			t.Errorf("inspector dragged over the sidebar: content %d wide, want 200", cw)
		}
		if sb2 := side.Unwrap().GetBounds(); sb2.Width != sb.Width {
			t.Errorf("the inspector drag changed the sidebar: %v, was %v", sb2, sb)
		}
	})
}

// Without a list in the content, macOS focuses an empty view, so no popup or button opens with a focus
// ring; elsewhere SWT keeps its choice. A content list and Focused win as in a sidebar window.
func TestInspectorFocus(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	sink := "popup"
	if runtime.GOOS == "darwin" {
		sink = "Composite"
	}
	for _, tc := range []struct {
		name    string
		content []gowt.Option
		want    string
	}{
		{"no list", []gowt.Option{gowt.Combo([]string{"Auto"}, gowt.ReadOnly(), name("popup")), gowt.Button("go", name("button"))}, sink},
		{"content table", []gowt.Option{gowt.Button("go"), gowt.Table(name("content table"))}, "content table"},
		{"focused", []gowt.Option{gowt.Combo([]string{"Auto"}, gowt.ReadOnly()), gowt.Text(name("text"), gowt.Focused())}, "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := "not checked"
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					w := app.Window("inspector")
					var content, insp *gowt.Panel
					w.Add(gowt.VBox(append([]gowt.Option{gowt.Ref(&content)}, tc.content...)...),
						gowt.VBox(gowt.Ref(&insp), gowt.Button("inspect")))
					look.SetInspector(w, content, insp, 240)
					w.Show()
					app.After(200*time.Millisecond, func() { got = focusName(app); w.Close() })
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("focus %s, want %s", got, tc.want)
			}
		})
	}
}
