package look_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/look"
)

// The order of SetToolbar and SetSidebar must not matter (the sidebar rebuilds the macOS toolbar), the
// toolbar buttons are ToolItem nodes that Click runs, and FocusSearch gives the search field the focus.
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
			if button.Class != "ToolItem" || button.Text != "New" || seg.Class != "ToolItem" || seg.Text != "Back" {
				t.Errorf("nodes %+v and %+v, want ToolItems titled New and Back", button, seg)
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
