package look_test

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/svg"
)

// The order of SetToolbar and SetSidebar must not matter (the sidebar rebuilds the macOS toolbar),
// Click runs the toolbar buttons, and FocusSearch gives the search field the focus.
func TestToolbarQA(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, toolbarFirst := range []bool{true, false} {
		name := "sidebar first"
		if toolbarFirst {
			name = "toolbar first"
		}
		t.Run(name, func(t *testing.T) {
			clicks, back := 0, 0
			var errs []string
			var button, seg qatree.Node
			var focused, searchFocused bool
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					w := app.Window("toolbar qa")
					var sidebar, content *gowt.Panel
					w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a"})), gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
					items := []look.ToolbarItem{
						{Title: "New", ID: "new", Symbol: "plus", OnClick: func() { clicks++ }},
						{Group: []look.ToolbarItem{{Title: "Back", ID: "back", Symbol: "chevron.backward", OnClick: func() { back++ }}}},
						{Title: "Search", ID: "search", OnSearch: func(string) {}},
					}
					if toolbarFirst {
						look.SetToolbar(w, items...)
						look.SetSidebar(w, sidebar, content, 200)
					} else {
						look.SetSidebar(w, sidebar, content, 200)
						look.SetToolbar(w, items...)
					}
					w.Show()
					app.After(400*time.Millisecond, func() {
						d := app.Unwrap()
						find := func(id string) qatree.Node {
							ns, ferr := qatree.Find(d, qatree.Selector{ID: id})
							if ferr != nil || len(ns) != 1 {
								errs = append(errs, "find "+id+": not one node")
								return qatree.Node{}
							}
							return ns[0]
						}
						button, seg = find("new"), find("back")
						for _, id := range []string{"new", "back", "new"} {
							if e := qatree.Act(d, qatree.Selector{ID: id}, "click", json.RawMessage(nil)); e != nil {
								errs = append(errs, "click "+id+": "+e.Error())
							}
						}
						focused = look.FocusSearch(w)
						until := time.Now().Add(2 * time.Second)
						var poll func()
						poll = func() {
							searchFocused = find("search").Focused
							if searchFocused || time.Now().After(until) {
								w.Close()
								return
							}
							app.After(50*time.Millisecond, poll)
						}
						poll()
					})
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range errs {
				t.Error(e)
			}
			if runtime.GOOS == "darwin" {
				if button.Class != "ToolItem" || button.Text != "New" || seg.Class != "ToolItem" || seg.Text != "Back" {
					t.Errorf("nodes %+v and %+v, want ToolItems titled New and Back", button, seg)
				}
			} else {
				if button.Class != "Canvas" || seg.Class != "Canvas" {
					t.Errorf("nodes %+v and %+v, want pill segments", button, seg)
				}
				for _, n := range []qatree.Node{button, seg} {
					// The segment canvas takes the ControlHeight square of its Size hint.
					if n.Bounds[2] != n.Bounds[3] || n.Bounds[2] <= 0 {
						t.Errorf("%s bounds %v, want a square of the control height", n.ID, n.Bounds)
					}
				}
			}
			if clicks != 2 || back != 1 {
				t.Errorf("clicks: New %d, Back %d, want 2 and 1", clicks, back)
			}
			if !focused || !searchFocused {
				t.Errorf("FocusSearch %v, the field has the focus in the dump: %v", focused, searchFocused)
			}
		})
	}
}

func TestToolbarOverflow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if runtime.GOOS == "darwin" {
		t.Skip("NSToolbar has its own overflow")
	}
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("toolbar overflow")
			var content, inspector *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&content)), gowt.VBox(gowt.Ref(&inspector)))
			look.SetInspector(w, content, inspector, 260)
			items := []look.ToolbarItem{{Title: "Back", ID: "back", Symbol: "chevron.backward", Nav: true},
				{Title: "Search", ID: "search", Small: true, OnSearch: func(string) {}}}
			// Seven actions fit within the 864 px toolbar available on the CI display;
			// sixteen could never all be restored there.
			for i := 0; i < 7; i++ {
				items = append(items, look.ToolbarItem{Title: fmt.Sprintf("Action %d", i), ID: fmt.Sprintf("action-%d", i), Symbol: "plus"})
			}
			doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>`)
			items = append(items,
				look.ToolbarItem{Title: "Image", ID: "image", Icon: app.ImageFromProvider(svg.NewImageDataProvider(doc, 16, 16, "#000000")), IconSVG: doc, Symbol: "plus"},
				look.ToolbarItem{Title: "SVG", ID: "svg", IconSVG: doc},
				look.ToolbarItem{Group: []look.ToolbarItem{{Title: "Forward", ID: "forward", Symbol: "chevron.forward"}}})
			clicks := 0
			enabled := rx.New(false)
			items = append(items,
				look.ToolbarItem{Space: true},
				look.ToolbarItem{Title: "Fit to Window", ID: "fit", Symbol: "arrow.up.left.and.arrow.down.right", Enabled: enabled, OnClick: func() { clicks++ }},
				look.ToolbarItem{Title: "Comment Mode", ID: "comment", Symbol: "text.bubble"})
			for i := 0; i < 8; i++ {
				items = append(items, look.ToolbarItem{Title: fmt.Sprintf("Inspector %d", i), ID: fmt.Sprintf("inspector-%d", i), Symbol: "plus", Inspector: true})
			}
			look.SetToolbar(w, items...)
			least := gowt.Scaled(160)
			look.SetInspectorWidth(w, least)
			// A size above the session's maximum tracking width can never settle,
			// so the wide phase must use the largest applicable width.
			wide := maxWindowWidth()
			if wide > 1600 {
				wide = 1600
			}
			w.SetSize(int(wide), 600)
			w.Show()
			d := app.Unwrap()
			check := func(step, class string) {
				for _, id := range []string{"fit", "comment"} {
					ns, findErr := qatree.Find(d, qatree.Selector{ID: id})
					if findErr != nil || len(ns) != 1 || ns[0].Class != class {
						fail("%s: %s is %+v (%v), want one %s", step, id, ns, findErr, class)
					}
					if id == "fit" && len(ns) == 1 && ns[0].Enabled != enabled.Peek() {
						fail("%s: fit enabled %v, want %v", step, ns[0].Enabled, enabled.Peek())
					}
				}
				bars, _ := qatree.Find(d, qatree.Selector{Class: "ToolBar"})
				for _, bar := range bars {
					if bar.Visible && bar.Bounds[3] != gowt.Spacing.ControlHeight {
						fail("%s: bar height %d, want %d", step, bar.Bounds[3], gowt.Spacing.ControlHeight)
					}
				}
				client := w.Unwrap().GetClientArea()
				for _, class := range []string{"Canvas", "ToolItem"} {
					buttons, _ := qatree.Find(d, qatree.Selector{Class: class})
					for _, button := range buttons {
						if button.Visible && int32(button.Bounds[0]+button.Bounds[2]) > client.X+client.Width {
							fail("%s: %s ends past the client area: %v", step, button.ID, button.Bounds)
						}
						// The segment canvas takes the ControlHeight square of its Size hint.
						if class == "Canvas" && button.Visible && (button.Bounds[2] != button.Bounds[3] || button.Bounds[2] <= 0) {
							fail("%s: %s bounds %v, want a square of the control height", step, button.ID, button.Bounds)
						}
					}
				}
			}
			// awaitSize runs f once the window has size width x height and rx bindings have caught
			// up. A fixed delay races a display that applies a resize in steps: the QA snapshot then
			// catches a mid-reflow overflow (items hidden for an intermediate width) or a state a
			// pending rx flush has not applied yet. Polling goes through the event loop, never inline:
			// work queued before the wait (resize messages, Async, rx effects) must land first. The
			// settle outcome and describe() are logged: a timeout must tell whether the shell never
			// reached the size (a display/size limit) or the reflow left the bar narrow at the final
			// width (items still hidden with room to spare).
			awaitSize := func(width, height int32, describe func() string, f func()) {
				start := time.Now()
				deadline := start.Add(5 * time.Second)
				logged := false
				var poll func()
				poll = func() {
					sz := w.Unwrap().GetSize()
					settled := sz.X == width && sz.Y == height && rx.Pending() == 0
					if !settled && time.Now().Before(deadline) {
						app.After(20*time.Millisecond, poll)
						return
					}
					if !logged {
						logged = true
						t.Logf("awaitSize %dx%d settled=%v after %s pending=%d %s", width, height, settled, time.Since(start).Round(time.Millisecond), rx.Pending(), describe())
					}
					f()
				}
				app.After(20*time.Millisecond, poll)
			}
			// diag is the geometry snapshot awaitSize logs with the settle outcome: the shell and
			// client area, the visible composites (the toolbar bar panel among them) and the overflow
			// menu population.
			diag := func() string {
				s := w.Unwrap()
				var comps []string
				if ns, ferr := qatree.Find(d, qatree.Selector{Class: "Composite"}); ferr == nil {
					for _, n := range ns {
						if n.Visible {
							comps = append(comps, fmt.Sprintf("%s@%v", n.Path, n.Bounds))
						}
					}
				}
				items, _ := qatree.Find(d, qatree.Selector{Class: "MenuItem"})
				return fmt.Sprintf("shell=%v client=%v inspector=%d menuItems=%d comps=%v %s", s.GetBounds(), s.GetClientArea(), look.InspectorWidth(w), len(items), comps, shellDiag(s))
			}
			// Find actual overflow instead of assuming a fixed width fits all
			// platform metrics and display limits.
			narrowWidth := func(f func(width int32)) {
				width := wide
				var step func()
				step = func() {
					w.SetSize(int(width), 600)
					awaitSize(width, 600, diag, func() {
						fit, _ := qatree.Find(d, qatree.Selector{ID: "fit"})
						if len(fit) == 1 && fit[0].Class == "MenuItem" {
							f(width)
							return
						}
						if width < 400 {
							fail("no width below %d overflowed fit, want one that does", wide)
							f(width)
							return
						}
						width -= 80
						step()
					})
				}
				step()
			}
			app.After(500*time.Millisecond, func() {
				narrowWidth(func(narrow int32) {
					check("narrow", "MenuItem")
					if got := look.InspectorWidth(w); got != least {
						fail("inspector width %d, want the requested minimum %d", got, least)
					}
					last, _ := qatree.Find(d, qatree.Selector{ID: "inspector-7"})
					if len(last) != 1 || last[0].Class != "MenuItem" {
						fail("last inspector command is %+v, want one overflow MenuItem", last)
					}
					if !look.FocusSearch(w) {
						fail("FocusSearch did not find the hidden field")
					}
					search, _ := qatree.Find(d, qatree.Selector{ID: "search"})
					if len(search) != 1 || search[0].Class != "Text" || !search[0].Focused {
						fail("revealed search is %+v, want one focused Text", search)
					} else {
						// Windows SEARCH reserves EM_SETMARGINS room for its search and clear icons and a
						// 2 px frame, so the outer control exceeds the 6*ControlHeight content hint; the hint
						// is the text area, not the bounds. The Small field is compact, so on Windows its
						// outer height stays within ControlHeight plus the native frame.
						if search[0].Bounds[2] < 6*gowt.Spacing.ControlHeight || search[0].Bounds[3] <= 0 {
							fail("search bounds %v, want a width of at least %d of text area and a positive height", search[0].Bounds, 6*gowt.Spacing.ControlHeight)
						}
						if runtime.GOOS == "windows" && search[0].Bounds[3] > gowt.Spacing.ControlHeight+2 {
							fail("search bounds %v, want the compact outer height within %d plus the native frame", search[0].Bounds, gowt.Spacing.ControlHeight)
						}
					}
					enabled.Set(true)
					if actErr := qatree.Act(d, qatree.Selector{ID: "fit"}, "click", json.RawMessage(nil)); actErr != nil {
						fail("click fit: %v", actErr)
					}
					if clicks != 1 {
						fail("fit clicks %d, want 1", clicks)
					}
					w.SetSize(int(wide), 600)
					awaitSize(wide, 600, diag, func() {
						check("wide", "Canvas")
						restored, _ := qatree.Find(d, qatree.Selector{ID: "forward"})
						if len(restored) != 1 || restored[0].Class != "Canvas" {
							fail("restored group segment is %+v, want one Canvas with its ID", restored)
						}
						w.SetSize(int(narrow), 600)
						awaitSize(narrow, 600, diag, func() {
							check("narrow again", "MenuItem")
							w.Close()
						})
					})
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

// The three-pane toolbar zones hold on macOS: the SidebarEnd item stands just past the sidebar
// edge, the Center items sit in the middle of the content column, a hidden sidebar moves the
// SidebarEnd item to the leading edge, and on a narrow window the content items overflow before
// the inspector items and come back when the window grows.
func TestToolbarZones(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("native NSToolbar zones; the bar zones are covered by the unit tests")
	}
	var errs []string
	fail := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("toolbar zones")
			var side, content, insp *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&side), gowt.Label("sidebar")),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Label("inspector")))
			look.SetSidebar(w, side, content, 220)
			look.SetInspector(w, content, insp, 260)
			nop := func() {}
			look.SetToolbar(w,
				look.ToolbarItem{Title: "Open", ID: "open", Symbol: "folder", Sidebar: true, OnClick: nop},
				look.ToolbarItem{Title: "Hide Sidebar", ID: "hide-sidebar", Symbol: "sidebar.left", SidebarEnd: true, OnClick: func() {
					look.SetSidebarVisible(w, !look.SidebarVisible(w))
				}},
				look.ToolbarItem{Title: "Zoom Out", ID: "zoom-out", Symbol: "minus.magnifyingglass", Center: true, OnClick: nop},
				look.ToolbarItem{Title: "Zoom In", ID: "zoom-in", Symbol: "plus.magnifyingglass", Center: true, OnClick: nop},
				look.ToolbarItem{Title: "Fit", ID: "fit", Symbol: "arrow.up.left.and.arrow.down.right", OnClick: nop},
				look.ToolbarItem{Title: "Info", ID: "info", Symbol: "info.circle", Inspector: true, OnClick: nop})
			w.SetSize(1000, 600)
			w.Show()
			d := app.Unwrap()
			node := func(step, id string) qatree.Node {
				ns, ferr := qatree.Find(d, qatree.Selector{ID: id})
				if ferr != nil || len(ns) != 1 {
					fail("%s: find %s: %v, %d nodes", step, id, ferr, len(ns))
					return qatree.Node{}
				}
				return ns[0]
			}
			// check verifies the zones against the sidebar and inspector edges of the window.
			check := func(step string, sidebar bool) {
				edge := int(w.Unwrap().GetClientArea().X)
				if sidebar {
					sb := side.Unwrap().GetBounds()
					edge = int(sb.X + sb.Width)
				}
				iedge := int(insp.Unwrap().GetBounds().X)
				hs := node(step, "hide-sidebar")
				if sidebar {
					// Just past the right edge of the sidebar, a bare icon before the title.
					if hs.Bounds[0] < edge || hs.Bounds[0] > edge+40 {
						fail("%s: hide-sidebar starts at %d, want just past the sidebar edge %d", step, hs.Bounds[0], edge)
					}
				} else if hs.Bounds[0] > 160 {
					fail("%s: hide-sidebar starts at %d, want at the leading edge with the sidebar hidden", step, hs.Bounds[0])
				}
				if !hs.Visible {
					fail("%s: hide-sidebar not visible", step)
				}
				out, in := node(step, "zoom-out"), node(step, "zoom-in")
				mid := (out.Bounds[0] + in.Bounds[0] + in.Bounds[2]) / 2
				want := (edge + iedge) / 2
				if mid < want-30 || mid > want+30 {
					fail("%s: center items mid %d, want about the content column middle %d", step, mid, want)
				}
				for _, id := range []string{"open", "zoom-out", "zoom-in", "fit"} {
					if n := node(step, id); n.Visible && n.Bounds[0]+n.Bounds[2] > iedge {
						fail("%s: %s ends at %d, past the inspector edge %d", step, id, n.Bounds[0]+n.Bounds[2], iedge)
					}
				}
				if n := node(step, "info"); n.Visible && n.Bounds[0] < iedge {
					fail("%s: info starts at %d, before the inspector edge %d", step, n.Bounds[0], iedge)
				}
			}
			app.After(1200*time.Millisecond, func() {
				check("start", true)
				// Hide the sidebar: the SidebarEnd item moves to the leading edge.
				look.SetSidebarVisible(w, false)
				app.After(1200*time.Millisecond, func() {
					check("hidden", false)
					look.SetSidebarVisible(w, true)
					// Narrow the window: the content items give way before the inspector items.
					w.SetSize(620, 600)
					app.After(1200*time.Millisecond, func() {
						nedge := int(insp.Unwrap().GetBounds().X)
						if n := node("narrow", "fit"); n.Visible && n.Bounds[0]+n.Bounds[2] > nedge {
							fail("narrow: fit crosses the inspector edge at %v", n.Bounds)
						}
						hidden := 0
						for _, id := range []string{"zoom-out", "zoom-in", "fit"} {
							if n := node("narrow", id); !n.Visible || n.Bounds[2] == 0 {
								hidden++
							}
						}
						if hidden == 0 {
							fail("narrow: no content item overflowed at 620 pt")
						}
						if n := node("narrow", "info"); !n.Visible || n.Bounds[0] < nedge {
							fail("narrow: info %v, want it on the inspector", n.Bounds)
						}
						w.SetSize(1000, 600)
						app.After(1200*time.Millisecond, func() {
							check("wide", true)
							for _, e := range errs {
								t.Error(e)
							}
							w.Close()
						})
					})
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
