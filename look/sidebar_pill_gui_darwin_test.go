package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// A Table and a Tree in a sidebar draw the selected row with the same row view, the Finder pill,
// which is never emphasized.
func TestSidebarTreePill(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	got := map[string]string{}
	emphasized := map[string]int64{}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("sidebar")
			var sidebar, content *gowt.Panel
			var table *gowt.TableView
			var tree *gowt.TreeView
			w.Add(
				gowt.VBox(gowt.Ref(&sidebar),
					gowt.Table(gowt.Ref(&table), gowt.Init(func(t *gowt.TableView) { t.Row("a"); t.Row("b") })),
					gowt.Tree(gowt.Ref(&tree), gowt.Init(func(t *gowt.TreeView) { t.Node("a"); t.Node("b") }))),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 200)
			table.Select(1)
			tree.Select(tree.Roots()[1])
			w.Show()
			app.After(300*time.Millisecond, func() {
				rowView := func(name string, view int64, row int64) {
					rv := cocoa.Msg(view, "rowViewAtRow:makeIfNecessary:", row, 1)
					if rv == 0 {
						got[name] = "no row view"
						return
					}
					got[name] = cocoa.OSClass_getName(cocoa.OSObject_getClass(rv))
					emphasized[name] = cocoa.Msg(rv, "isEmphasized") & 0xff
				}
				rowView("table", table.Unwrap().View.Id, 1)
				rowView("tree", tree.Unwrap().View.Id, 1)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"table", "tree"} {
		if got[name] != "GowtSidebarRowView" || emphasized[name] != 0 {
			t.Errorf("%s: row view %q, emphasized %d; want GowtSidebarRowView, 0", name, got[name], emphasized[name])
		}
	}
}
