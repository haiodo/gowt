package gowt

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// steps runs f one after another, 250 ms apart, so that the event loop lays out between them: GTK sizes the
// rows of a tree at idle time, and a change of a native size shows only a turn later.
func steps(app *App, f ...func()) {
	var next func(i int)
	next = func(i int) {
		app.After(250*time.Millisecond, func() {
			f[i]()
			if i+1 < len(f) {
				next(i + 1)
			}
		})
	}
	next(0)
}

// A Tree with MaxHeight is as high as its rows until the cap, stops there when a node with 100
// children opens, follows a collapse, and leaves the Table above it its height.
func TestMaxHeightTree(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []error
	got := map[string]int{}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("maxheight")
			w.SetSize(300, 700)
			var table *TableView
			var tree *TreeView
			var root *Node
			w.Add(VBox(Padding(0), Grow(),
				Table(Ref(&table), Grow(), Size(0, 120)),
				Tree(Ref(&tree), GrowX(), MaxHeight(200))))
			root = tree.Node("root")
			tree.Node("second")
			for i := range 100 {
				root.Node(fmt.Sprint("child ", i))
			}
			w.Show()
			h := func(v Widget) int { return int(v.control().GetBounds().Height) }
			steps(app,
				func() {
					got["closed"] = h(tree)
					got["table"] = h(table)
					root.SetExpanded(true)
				},
				func() {
					got["open"] = h(tree)
					got["tableOpen"] = h(table)
					root.SetExpanded(false)
				},
				func() {
					got["closedAgain"] = h(tree)
					root.SetExpanded(true)
				},
				func() {
					tree.SetMaxHeight(0)
				},
				func() {
					got["noCap"] = h(tree)
					tree.SetMaxHeight(150)
				},
				func() {
					got["cap150"] = h(tree)
					tree.SetSize(0, 90)
				},
				func() {
					got["size90"] = h(tree)
					w.Close()
				})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	t.Logf("%v", got)
	if got["closed"] <= 0 || got["closed"] >= 200 {
		t.Errorf("two roots: %d high, want natural, below the cap 200", got["closed"])
	}
	if got["open"] != 200 {
		t.Errorf("102 rows: %d high, want the cap 200", got["open"])
	}
	if got["tableOpen"] < 100 {
		t.Errorf("Table pushed out: %d high with the Tree open, %d before", got["tableOpen"], got["table"])
	}
	if got["closedAgain"] != got["closed"] {
		t.Errorf("collapsed: %d high, was %d", got["closedAgain"], got["closed"])
	}
	if got["noCap"] <= 200 {
		t.Errorf("no cap: %d high, want the natural height of 102 rows", got["noCap"])
	}
	if got["cap150"] != 150 {
		t.Errorf("cap 150: %d high", got["cap150"])
	}
	// A height hint is the client height, as with Size: the border comes on top.
	if got["size90"] < 90 || got["size90"] > 120 {
		t.Errorf("SetSize(0, 90): %d high", got["size90"])
	}
}

// The window is lower than the Table's minimum and the Tree's natural height together: the Table keeps
// its Size as a minimum, the Tree takes what is left and scrolls, and nothing runs out of the window.
func TestMaxHeightShortWindow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []error
	got := map[string]int{}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("shortwindow")
			w.SetSize(300, 300)
			var table *TableView
			var tree *TreeView
			var root *Node
			w.Add(VBox(Padding(0), Gap(0), Grow(),
				Table(Ref(&table), Grow(), Size(0, 120)),
				Tree(Ref(&tree), GrowX(), MaxHeight(400))))
			root = tree.Node("root")
			for i := range 100 {
				root.Node(fmt.Sprint("child ", i))
			}
			w.Show()
			read := func(name string) {
				got[name+"Table"] = int(table.control().GetBounds().Height)
				got[name+"Tree"] = int(tree.control().GetBounds().Height)
				got[name+"TreeY"] = int(tree.control().GetBounds().Y)
				got[name+"Client"] = int(w.Unwrap().GetClientArea().Height)
			}
			steps(app,
				func() { root.SetExpanded(true) },
				func() { read("short"); w.SetSize(300, 900) },
				func() { read("tall"); w.SetSize(300, 300) },
				func() { read("short2"); w.Close() })
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	t.Logf("%v", got)
	for _, k := range []string{"short", "short2"} {
		if got[k+"Table"] < 120 {
			t.Errorf("%s: Table %d high, want its minimum 120", k, got[k+"Table"])
		}
		if got[k+"Tree"] < 40 || got[k+"TreeY"]+got[k+"Tree"] > got[k+"Client"] {
			t.Errorf("%s: Tree %d high at y %d in a client area %d high", k, got[k+"Tree"], got[k+"TreeY"], got[k+"Client"])
		}
	}
	if got["tallTree"] != 400 {
		t.Errorf("tall window: Tree %d high, want its cap 400", got["tallTree"])
	}
}

// Expanding a node lays the parents out again by itself, as does adding a node; no Relayout is needed.
func TestTreeRelayoutOnExpand(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []error
	got := map[string]int{}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("treerelayout")
			w.SetSize(300, 700)
			var tree *TreeView
			var root *Node
			w.Add(VBox(Padding(0), Grow(),
				Label("top", GrowX()),
				Tree(Ref(&tree), GrowX())))
			root = tree.Node("root")
			for i := range 5 {
				root.Node(fmt.Sprint("child ", i))
			}
			w.Show()
			h := func() int { return int(tree.control().GetBounds().Height) }
			steps(app,
				func() {
					got["closed"] = h()
					root.SetExpanded(true)
				},
				func() {
					got["open"] = h()
					root.Node("one more")
				},
				func() {
					got["added"] = h()
					root.Remove()
				},
				func() {
					got["removed"] = h()
					w.Close()
				})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	t.Logf("%v", got)
	if got["open"] <= got["closed"] || got["added"] <= got["open"] || got["removed"] >= got["added"] {
		t.Errorf("heights do not follow the rows: %v", got)
	}
}

// Deselect clears a List, a Table and a Tree and runs no OnSelect, as SWT's deselectAll sends no Selection.
func TestDeselect(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	var after [3]int
	events, base := map[string]int{}, map[string]int{}
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("deselect")
			var list *ListView
			var table *TableView
			var tree *TreeView
			w.Add(VBox(
				List([]string{"a", "b"}, Ref(&list), OnSelect(func(int) { events["list"]++ })),
				Table(Ref(&table), OnSelect(func(int) { events["table"]++ })),
				Tree(Ref(&tree), OnSelect(func(*Node) { events["tree"]++ }))))
			table.Column("c", 80)
			table.Row("a")
			table.Row("b")
			a := tree.Node("a")
			tree.Node("b")
			w.Show()
			app.After(200*time.Millisecond, func() {
				list.Select(1)
				table.Select(1)
				tree.Select(a)
				app.Flush()
				if len(list.Selected()) != 1 || len(table.Selected()) != 1 || len(tree.Selected()) != 1 {
					t.Errorf("selected before: %v %v %v", list.Selected(), table.Selected(), len(tree.Selected()))
				}
				for k, v := range events {
					base[k] = v
				}
				list.Deselect()
				table.Deselect()
				tree.Deselect()
				app.Flush()
				after = [3]int{len(list.Selected()), len(table.Selected()), len(tree.Selected())}
				tree.Select(a)
				tree.Select(nil)
				if n := len(tree.Selected()); n != 0 {
					t.Errorf("Select(nil): %d selected", n)
				}
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if after != [3]int{} {
		t.Errorf("selected after Deselect: %v", after)
	}
	// Select may report on some platforms; Deselect must add nothing to what it reported.
	for _, k := range []string{"list", "table", "tree"} {
		if events[k] != base[k] {
			t.Errorf("Deselect called OnSelect of the %s: %d calls, %d before", k, events[k], base[k])
		}
	}
}
