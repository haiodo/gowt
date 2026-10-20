package look_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

func init() { runtime.LockOSThread() }

var mainq = make(chan func())

// The toolkit needs the main thread: the tests run on their own goroutine and hand the UI work back.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func onMain(f func()) {
	ok := make(chan struct{})
	mainq <- func() { defer close(ok); f() }
	<-ok
}

// A list selected before SetSidebar reported its row as a new selection on macOS.
func TestSidebarKeepsSelectionQuiet(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var events []string
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar")
			var sidebar, content *gowt.Panel
			var table *gowt.TableView
			var list *gowt.ListView
			var tree *gowt.TreeView
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar),
					gowt.Table(gowt.Ref(&table), gowt.Init(func(t *gowt.TableView) { t.Row("a"); t.Row("b") })),
					gowt.List([]string{"a", "b"}, gowt.Ref(&list)),
					gowt.Tree(gowt.Ref(&tree), gowt.Init(func(t *gowt.TreeView) { t.Node("a"); t.Node("b") }))),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			table.Select(1)
			list.Select(1)
			tree.Select(tree.Roots()[1])
			table.OnSelect(func(i int) { events = append(events, fmt.Sprint("table ", i)) })
			list.OnSelect(func(i int) { events = append(events, fmt.Sprint("list ", i)) })
			tree.OnSelect(func(n *gowt.Node) { events = append(events, "tree "+n.Text()) })
			look.SetSidebar(w, sidebar, content, 200)
			w.Show()
			app.After(300*time.Millisecond, w.Close)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) > 0 {
		t.Errorf("SetSidebar sent Selection events: %v", events)
	}
}

// Where the toolbar is a bar of gowt widgets, a Sidebar item goes into a row at the top of the
// sidebar, not into the bar above the content.
func TestSidebarItemRow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var native bool
	var items int
	var rowY, listY int32
	onMain(func() {
		err := gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar")
			var sidebar, content *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a"})), gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 200)
			native = look.SetToolbar(w, look.ToolbarItem{Title: "New", Symbol: "plus", Sidebar: true},
				look.ToolbarItem{Title: "Share", Symbol: "square.and.arrow.up"})
			w.Show()
			kids := sidebar.Unwrap().GetChildren()
			row := kids[0]
			rowY, listY = row.GetBounds().Y, kids[len(kids)-1].GetBounds().Y
			if c, ok := row.Impl().(*swt.Composite); ok {
				for _, ch := range c.GetChildren() {
					if tb, ok := ch.Impl().(*swt.ToolBar); ok {
						items += int(tb.GetItemCount())
					}
				}
			}
			app.After(300*time.Millisecond, w.Close)
		})
		if err != nil {
			t.Error(err)
		}
	})
	if native {
		t.Skip("native toolbar")
	}
	if items != 1 {
		t.Errorf("sidebar row has %d tool items, want 1", items)
	}
	if rowY >= listY {
		t.Errorf("sidebar row at y %d, not above the list at y %d", rowY, listY)
	}
}

// Without gowt.Focused a sidebar window opens with the focus on the first list of the content, then of
// the sidebar, not on the button before it; Focused still wins.
func TestSidebarFocus(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, tc := range []struct {
		name    string
		content []gowt.Option
		want    string
	}{
		{"sidebar list", []gowt.Option{gowt.Label("content")}, "sidebar list"},
		{"content table", []gowt.Option{gowt.Button("open"), gowt.Table(name("content table"))}, "content table"},
		{"focused", []gowt.Option{gowt.Table(), gowt.Text(name("text"), gowt.Focused())}, "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					w := app.Window("sidebar")
					var sidebar, content *gowt.Panel
					w.Add(
						gowt.VBox(gowt.Ref(&sidebar), gowt.Button("new"), gowt.List([]string{"a"}, name("sidebar list"))),
						gowt.VBox(append([]gowt.Option{gowt.Ref(&content)}, tc.content...)...))
					look.SetSidebar(w, sidebar, content, 200)
					w.Show()
					// On macOS the first window of the process becomes key only after Show.
					if app.Unwrap().GetActiveShell() == w.Unwrap() {
						got = append(got, focusName(app))
					}
					app.After(200*time.Millisecond, func() { got = append(got, focusName(app)); w.Close() })
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, g := range got {
				if g != tc.want {
					t.Errorf("focus %v, want %s", got, tc.want)
					break
				}
			}
		})
	}
}

func name(s string) gowt.Option {
	return gowt.Custom(func(c *swt.Control) { c.SetDataKeyValue("name", s) })
}

func focusName(app *gowt.App) string {
	c := app.Unwrap().GetFocusControl()
	if c == nil {
		return "nothing"
	}
	if s, ok := c.GetDataKey("name").(string); ok {
		return s
	}
	return c.GetName()
}

// A Table without columns, header and grid lines is the row-with-icon list of a sidebar (a List row has
// no image). Its rows must not be lower than a List's, show the image before the text, and take the arrow keys.
func TestSidebarIconTable(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var listH, tableH, rowH, imgX, textX, textW int32
	var imgW int32
	var sel []int
	var blocked string
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar")
			var sidebar, content *gowt.Panel
			var list *gowt.ListView
			var table *gowt.TableView
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar), gowt.Padding(0), gowt.Gap(0),
					gowt.List([]string{"a", "b", "c"}, gowt.NoBorder(), gowt.Size(0, 100), gowt.Ref(&list)),
					gowt.Table(gowt.NoBorder(), gowt.Grow(), gowt.Ref(&table), gowt.Init(func(tv *gowt.TableView) {
						tv.ShowHeader(false)
						tv.ShowLines(false)
						for _, s := range []string{"a", "b", "c"} {
							tv.Row(s).SetImageAt(0, icons.Get(app, "file"))
						}
					}))),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 240)
			w.Show()
			listH = list.Unwrap().GetItemHeight()
			tableH = table.Unwrap().GetItemHeight()
			item := table.Unwrap().GetItem(1)
			rowH = item.GetBounds().Height
			imgX, imgW = item.GetImageBounds(0).X, item.GetImageBounds(0).Width
			textX, textW = item.GetTextBounds(0).X, item.GetTextBounds(0).Width
			table.OnSelect(func(i int) { sel = append(sel, i) })
			table.Select(0)
			tbl := table.Unwrap()
			tbl.SetFocus()
			app.After(200*time.Millisecond, func() {
				if blocked = postBlocked(app); blocked != "" {
					w.Close()
					return
				}
				for _, k := range []int32{swt.ARROW_DOWN, swt.ARROW_DOWN, swt.ARROW_UP} {
					ev := swt.NewEvent()
					ev.Type, ev.KeyCode = swt.KeyDown, k
					app.Unwrap().Post(ev)
					ev.Type = swt.KeyUp
					app.Unwrap().Post(ev)
				}
				app.After(500*time.Millisecond, func() {
					sel = append(sel, table.Index())
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("item height list %d table %d, row bounds height %d, image x %d w %d, text x %d w %d, selection %v",
		listH, tableH, rowH, imgX, imgW, textX, textW, sel)
	// An icon can make a row taller than a text-only one (Windows: 19 against 15), never shorter.
	if tableH < listH {
		t.Errorf("table row height %d, list row height %d", tableH, listH)
	}
	if imgW > 0 && textX < imgX+imgW {
		t.Errorf("text at x %d overlaps the image at x %d, width %d", textX, imgX, imgW)
	}
	if textW <= 0 {
		t.Errorf("text bounds width %d", textW)
	}
	if blocked != "" {
		t.Log("arrow keys not checked: " + blocked)
	} else if len(sel) == 0 || sel[len(sel)-1] != 1 {
		t.Errorf("down, down, up from row 0: selection %v, want it to end at 1", sel)
	}
}

// sidebarWindow opens a sidebar window of 900 by 600 points with a toolbar of items, or by default
// one whose first item is over the sidebar, and runs check once it is on screen.
func sidebarWindow(t *testing.T, items []look.ToolbarItem, check func(app *gowt.App, w *gowt.Window, sidebar, content *gowt.Panel, native bool)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar")
			var sidebar, content *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a", "b"}, gowt.Grow())),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 220)
			if items == nil {
				items = []look.ToolbarItem{{Title: "Sidebar", Symbol: "sidebar.left", Sidebar: true},
					{Title: "Share", Symbol: "square.and.arrow.up"}}
			}
			native := look.SetToolbar(w, items...)
			w.SetSize(900, 600)
			w.Show()
			app.After(300*time.Millisecond, func() {
				defer w.Close()
				check(app, w, sidebar, content, native)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// barLead is the first control of the toolbar bar above the content, nil with a native toolbar.
func barLead(w *gowt.Window, sidebar, content *gowt.Panel) *swt.Control {
	for _, c := range w.Unwrap().GetChildren() {
		p, ok := c.Impl().(*swt.Composite)
		if !ok || p == sidebar.Unwrap() || p == content.Unwrap() || len(p.GetChildren()) == 0 {
			continue
		}
		return p.GetChildren()[0]
	}
	return nil
}

func TestSidebarHide(t *testing.T) {
	sidebarWindow(t, nil, func(app *gowt.App, w *gowt.Window, sidebar, content *gowt.Panel, native bool) {
		var changes []string
		look.OnSidebarChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		r := w.Unwrap().GetClientArea()
		shownX := content.Unwrap().GetBounds().X
		if !look.SidebarVisible(w) || look.SidebarWidth(w) != 220 || shownX < 220 {
			t.Errorf("start: visible %v, width %d, content x %d", look.SidebarVisible(w), look.SidebarWidth(w), shownX)
		}
		lead := barLead(w, sidebar, content)
		if !native && (lead == nil || lead.GetVisible()) {
			t.Errorf("bar starts with a visible Sidebar item copy: %v", lead)
		}

		look.SetSidebarVisible(w, false)
		app.Flush()
		b := content.Unwrap().GetBounds()
		if look.SidebarVisible(w) || sidebar.Unwrap().GetVisible() || b.X != r.X || b.Width != r.Width {
			t.Errorf("hidden: visible %v, sidebar shown %v, content %v in client area %v",
				look.SidebarVisible(w), sidebar.Unwrap().GetVisible(), b, r)
		}
		if !native && (lead == nil || !lead.GetVisible()) {
			t.Error("hidden: the Sidebar item did not move to the bar")
		}
		look.SetSidebarVisible(w, false) // no change, no callback

		look.SetSidebarVisible(w, true)
		look.SetSidebarWidth(w, 50)
		app.Flush()
		// The narrowest sidebar is 120 points or the room of the sidebar.left item, which on macOS is more.
		least := int32(look.SidebarWidth(w))
		t.Logf("narrowest sidebar with one Sidebar item: %d", least)
		if least < 120 || least >= 220 || content.Unwrap().GetBounds().X != shownX-(220-least) {
			t.Errorf("width 50: got %d, content x %d, want 120 to 219 and x %d", least, content.Unwrap().GetBounds().X, shownX-(220-least))
		}
		if want := fmt.Sprintf("[false 220 true 220 true %d]", least); fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
	})
}

func sashOf(w *gowt.Window) *swt.Sash {
	for _, c := range w.Unwrap().GetChildren() {
		if s, ok := c.Impl().(*swt.Sash); ok {
			return s
		}
	}
	return nil
}

// A drag delivered as the events a Sash sends: Selection with the new sash x between the mouse
// down and up. The pointer does not move; TestSidebarPostedDrag moves it.
func TestSidebarDragEvents(t *testing.T) {
	sidebarWindow(t, nil, func(app *gowt.App, w *gowt.Window, sidebar, content *gowt.Panel, native bool) {
		var changes []string
		look.OnSidebarChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		sash := sashOf(w)
		if sash == nil {
			t.Error("no sash")
			return
		}
		x0 := sash.GetBounds().X
		send := func(typ int32, x int32) int32 {
			e := swt.NewEvent()
			e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
			sash.NotifyListeners(typ, e)
			return e.X
		}
		send(swt.MouseDown, 0)
		if got := send(swt.Selection, x0+80); got != x0+80 || look.SidebarWidth(w) != 300 {
			t.Errorf("drag by 80: sash at %d, width %d", got, look.SidebarWidth(w))
		}
		if len(changes) != 0 {
			t.Errorf("changes during the drag: %v", changes)
		}
		send(swt.Selection, 20)
		least := look.SidebarWidth(w)
		if !look.SidebarVisible(w) || least < 120 || least >= 300 {
			t.Errorf("dragged to 20: visible %v, width %d; want the drag to stop at the narrowest width", look.SidebarVisible(w), least)
		}
		send(swt.MouseUp, 0)
		app.Flush()
		if want := fmt.Sprintf("[true %d]", least); fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
		r := w.Unwrap().GetClientArea()
		send(swt.MouseDown, 0)
		send(swt.Selection, r.Width)
		send(swt.MouseUp, 0)
		if !look.SidebarVisible(w) || look.SidebarWidth(w) > int(r.Width)-200 {
			t.Errorf("dragged to the right edge: visible %v, width %d of %d", look.SidebarVisible(w), look.SidebarWidth(w), r.Width)
		}
		look.SetSidebarVisible(w, false)
		if sash.GetVisible() {
			t.Error("a hidden sidebar keeps its sash")
		}
	})
}

// A real drag of the sidebar edge by 60 points, through the system's input (postMouse).
func TestSidebarPostedDrag(t *testing.T) {
	skip := ""
	sidebarWindow(t, nil, func(app *gowt.App, w *gowt.Window, sidebar, content *gowt.Panel, native bool) {
		if skip = postBlocked(app); skip != "" {
			return
		}
		sash := sashOf(w)
		b := sash.GetBounds()
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
		look.OnSidebarChange(w, func(visible bool, width int) { changes = append(changes, fmt.Sprint(visible, width)) })
		ok := post(swt.MouseMove, p.X) && post(swt.MouseDown, p.X)
		for i := int32(1); ok && i <= 6; i++ {
			ok = post(swt.MouseMove, p.X+10*i)
		}
		ok = ok && post(swt.MouseUp, p.X+60)
		if !ok {
			skip = "the mouse events could not be posted"
			return
		}
		if got := look.SidebarWidth(w); got != 280 {
			t.Errorf("drag by 60: width %d, want 280", got)
		}
		if want := "[true 280]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
	})
	if skip != "" {
		t.Skip(skip)
	}
}

// Sidebar items wider than 120 points set the narrowest sidebar: SetSidebarWidth and a drag stop
// there, so the items never cross into the content, and the drag does not hide the sidebar.
func TestSidebarItemsMinWidth(t *testing.T) {
	var items []look.ToolbarItem
	for _, s := range []string{"Scan Node.js", "Scan Docker", "Open Profile..."} {
		items = append(items, look.ToolbarItem{Title: s, Sidebar: true})
	}
	sidebarWindow(t, items, func(app *gowt.App, w *gowt.Window, sidebar, content *gowt.Panel, native bool) {
		look.SetSidebarWidth(w, 500)
		app.Flush()
		look.SetSidebarWidth(w, 120)
		least := look.SidebarWidth(w)
		t.Logf("narrowest sidebar for three text items: %d", least)
		if least <= 200 {
			t.Errorf("SetSidebarWidth(120) gave %d, too narrow for three text items", least)
		}
		sash := sashOf(w)
		send := func(typ int32, x int32) int32 {
			e := swt.NewEvent()
			e.X, e.Y, e.Button = x, sash.GetBounds().Y, 1
			sash.NotifyListeners(typ, e)
			return e.X
		}
		x0 := sash.GetBounds().X
		send(swt.MouseDown, 0)
		got := send(swt.Selection, x0-60)
		send(swt.MouseUp, 0)
		if !look.SidebarVisible(w) || look.SidebarWidth(w) != least || got != x0 {
			t.Errorf("dragged 60 points below the narrowest %d: visible %v, width %d, sash at %d, want it to stay at %d",
				least, look.SidebarVisible(w), look.SidebarWidth(w), got, x0)
		}
	})
}
