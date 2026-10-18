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
