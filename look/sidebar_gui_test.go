package look_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/look"
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
			table.SetSelection(1)
			list.SetSelection(1)
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
