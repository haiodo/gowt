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
)

// Small macOS toolbar items: the QA driver finds and clicks a Small symbol item, a Small text item,
// a Sidebar item, a group segment and the search field. Drawn icon sizes are checked in a real
// window by internal/cocoa TestToolbarSmallSymbolInWindow.
func TestToolbarSmallItems(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("toolbar small")
			var sidebar, content *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a"})), gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			clicks := 0
			items := []look.ToolbarItem{
				{Title: "Add", ID: "add", Symbol: "plus", OnClick: func() { clicks++ }},
				{Title: "Small Add", ID: "small-add", Symbol: "plus", Small: true, OnClick: func() { clicks++ }},
				{Title: "Zoom", ID: "zoom", Symbol: "chevron.up", Small: true, Sidebar: true, OnClick: func() { clicks++ }},
				{Title: "100%", ID: "zoom-text", Text: rx.New("100%"), Small: true, OnClick: func() { clicks++ }},
				{Group: []look.ToolbarItem{{Title: "Back", ID: "back", Symbol: "chevron.backward", OnClick: func() { clicks++ }}}},
				{Title: "Search", ID: "search", Small: true, OnSearch: func(string) {}},
			}
			look.SetSidebar(w, sidebar, content, 200)
			look.SetToolbar(w, items...)
			w.Show()
			app.After(400*time.Millisecond, func() {
				d := app.Unwrap()
				find := func(id string) qatree.Node {
					ns, ferr := qatree.Find(d, qatree.Selector{ID: id})
					if ferr != nil || len(ns) != 1 {
						fail("find %s: %+v (%v), want one node", id, ns, ferr)
						return qatree.Node{}
					}
					return ns[0]
				}
				text := find("zoom-text")
				seg, search := find("back"), find("search")
				for _, id := range []string{"add", "small-add", "zoom", "zoom-text", "back"} {
					if e := qatree.Act(d, qatree.Selector{ID: id}, "click", json.RawMessage(nil)); e != nil {
						fail("click %s: %v", id, e)
					}
				}
				if clicks != 5 {
					fail("clicks %d, want 5", clicks)
				}
				if seg.Class != "ToolItem" || search.Class != "Text" {
					fail("group segment %+v and search %+v keep their classes", seg, search)
				}
				if text.Class != "ToolItem" || text.Text != "100%" {
					fail("small text item %+v, want a ToolItem titled 100%%", text)
				}
				// The drawn icon sizes and the sidebar capsule are AppKit internals of the item views;
				// internal/cocoa TestToolbarSmallSymbolInWindow checks them in a real window.
				w.Close()
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
