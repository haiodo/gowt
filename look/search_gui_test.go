package look_test

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// The toolbar search field is a native NSSearchField on macOS and a gowt Text in the bar elsewhere;
// both are a Text in the QA dump.
func TestToolbarSearch(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var searches []string
	var errs []string
	var noToolbar, focused bool
	var node qatree.Node
	var got []string
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			plain := app.Window("plain")
			noToolbar = look.FocusSearch(plain)
			plain.Close()
			w := app.Window("search")
			var sidebar, content *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a"})), gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 200)
			look.SetToolbar(w, look.ToolbarItem{Title: "Search", ID: "search", OnSearch: func(s string) { searches = append(searches, s) }})
			w.Show()
			app.After(400*time.Millisecond, func() {
				focused = look.FocusSearch(w)
				d := app.Unwrap()
				sel := qatree.Selector{ID: "search"}
				ns, ferr := qatree.Find(d, sel)
				if ferr != nil || len(ns) != 1 {
					errs = append(errs, "find: not one node")
					w.Close()
					return
				}
				node = ns[0]
				act := func(action, args string) {
					if e := qatree.Act(d, sel, action, json.RawMessage(args)); e != nil {
						errs = append(errs, action+": "+e.Error())
					}
				}
				act("type", `{"text":"abc"}`)
				got = append(got, text(d, sel))
				searches = nil
				act("key", `{"keys":"Enter"}`)
				if runtime.GOOS == "darwin" {
					act("key", `{"keys":"Esc"}`)
					got = append(got, text(d, sel))
				}
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if noToolbar {
		t.Error("FocusSearch of a window with no toolbar reported true")
	}
	if !focused {
		t.Error("FocusSearch reported false")
	}
	if node.Class != "Text" || node.Text != "Search" {
		t.Errorf("dump node %+v, want a Text with the placeholder Search", node)
	}
	if runtime.GOOS != "darwin" && !node.Focused {
		t.Error("the search field does not have the focus after FocusSearch")
	}
	for _, e := range errs {
		t.Error(e)
	}
	if len(got) == 0 || got[0] != "abc" {
		t.Errorf("text after Type: %q, want abc", got)
	}
	if len(searches) == 0 || searches[0] != "abc" {
		t.Errorf("OnSearch after Enter: %q, want it to start with abc", searches)
	}
	if runtime.GOOS == "darwin" && (len(got) != 2 || got[1] != "" || searches[len(searches)-1] != "") {
		t.Errorf("after Esc: text %q, searches %q, want the field cleared and OnSearch with an empty text", got, searches)
	}
}

func text(d *swt.Display, s qatree.Selector) string {
	ns, err := qatree.Find(d, s)
	if err != nil || len(ns) != 1 {
		return "?"
	}
	return ns[0].Value
}
