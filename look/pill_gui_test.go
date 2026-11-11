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
)

// The icon items of the widget bar are pill segments: one canvas per segment, square at the control
// height, the segments of a group touching as one capsule, and a pointer press on a segment runs its
// click. The native macOS toolbar draws its own items and never builds the widget bar.
func TestToolbarPills(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if runtime.GOOS == "darwin" {
		t.Skip("the macOS toolbar is native")
	}
	var errs []string
	fail := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	var newClicks, backClicks, fwdClicks, dimClicks int
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("pills")
			var sidebar, content *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.List([]string{"a"})),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")))
			look.SetSidebar(w, sidebar, content, 220)
			enabled := rx.New(true)
			look.SetToolbar(w,
				look.ToolbarItem{Title: "New", ID: "new", Symbol: "plus", OnClick: func() { newClicks++ }},
				look.ToolbarItem{Group: []look.ToolbarItem{
					{Title: "Back", ID: "back", Symbol: "chevron.backward", OnClick: func() { backClicks++ }},
					{Title: "Forward", ID: "forward", Symbol: "chevron.forward", OnClick: func() { fwdClicks++ }},
				}},
				look.ToolbarItem{Title: "Dim", ID: "dim", Symbol: "ellipsis", Enabled: enabled, OnClick: func() { dimClicks++ }},
				look.ToolbarItem{Title: "Share", ID: "share", Symbol: "square.and.arrow.up"})
			w.Show()
			app.After(400*time.Millisecond, func() {
				d := app.Unwrap()
				node := func(id string) qatree.Node {
					ns, ferr := qatree.Find(d, qatree.Selector{ID: id})
					if ferr != nil || len(ns) != 1 {
						fail("find %s: %v, %d nodes", id, ferr, len(ns))
						return qatree.Node{}
					}
					if ns[0].Class != "Canvas" {
						fail("%s is a %s, want a pill segment canvas", id, ns[0].Class)
					}
					return ns[0]
				}
				newN, backN, fwdN, dimN, shareN := node("new"), node("back"), node("forward"), node("dim"), node("share")
				for _, n := range []qatree.Node{newN, backN, fwdN, dimN, shareN} {
					// The height is stable: the hover and press fills never resize the button.
					if n.Bounds[3] != gowt.Spacing.ControlHeight {
						fail("%s height %d, want the control height %d", n.ID, n.Bounds[3], gowt.Spacing.ControlHeight)
					}
				}
				// The group is one capsule: the segments touch, no gap between them.
				if backN.Bounds[0]+backN.Bounds[2] != fwdN.Bounds[0] {
					fail("back ends at %d, forward starts at %d, want them touching", backN.Bounds[0]+backN.Bounds[2], fwdN.Bounds[0])
				}
				for _, id := range []string{"new", "back", "forward", "dim"} {
					if e := qatree.Act(d, qatree.Selector{ID: id}, "click", json.RawMessage(nil)); e != nil {
						fail("click %s: %v", id, e)
					}
				}
				enabled.Set(false)
				app.Flush()
				ns, _ := qatree.Find(d, qatree.Selector{ID: "dim"})
				if len(ns) == 1 && ns[0].Enabled {
					fail("dim is still enabled after Enabled turned false")
				}
				// Keyboard parity with the Button the pill replaced: a focused segment takes Space
				// and Enter, a disabled one nothing.
				keys := func(id, k string) {
					if e := qatree.Act(d, qatree.Selector{ID: id}, "focus", json.RawMessage(nil)); e != nil {
						fail("focus %s: %v", id, e)
					}
					if e := qatree.Act(d, qatree.Selector{ID: id}, "key", json.RawMessage(`{"keys":"`+k+`"}`)); e != nil {
						fail("key %s %s: %v", id, k, e)
					}
				}
				keys("new", "space")
				keys("back", "enter")
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
	if newClicks != 2 || backClicks != 2 || fwdClicks != 1 || dimClicks != 1 {
		t.Errorf("clicks: new %d, back %d, forward %d, dim %d, want 2, 2, 1, 1", newClicks, backClicks, fwdClicks, dimClicks)
	}
}
