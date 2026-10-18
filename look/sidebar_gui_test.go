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
					got = append(got, focusName(app))
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
