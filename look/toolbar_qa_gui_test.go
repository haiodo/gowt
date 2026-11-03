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
	"github.com/haiodo/gowt/svg"
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

func TestToolbarOverflow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if runtime.GOOS == "darwin" {
		t.Skip("NSToolbar has its own overflow")
	}
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("toolbar overflow")
			var content, inspector *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&content)), gowt.VBox(gowt.Ref(&inspector)))
			look.SetInspector(w, content, inspector, 260)
			items := []look.ToolbarItem{{Title: "Back", ID: "back", Symbol: "chevron.backward", Nav: true},
				{Title: "Search", ID: "search", OnSearch: func(string) {}}}
			for i := 0; i < 16; i++ {
				items = append(items, look.ToolbarItem{Title: fmt.Sprintf("Action %d", i), ID: fmt.Sprintf("action-%d", i), Symbol: "plus"})
			}
			doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>`)
			items = append(items,
				look.ToolbarItem{Title: "Image", ID: "image", Icon: app.ImageFromProvider(svg.NewImageDataProvider(doc, 16, 16, "#000000")), IconSVG: doc, Symbol: "plus"},
				look.ToolbarItem{Title: "SVG", ID: "svg", IconSVG: doc})
			clicks := 0
			enabled := rx.New(false)
			items = append(items,
				look.ToolbarItem{Space: true},
				look.ToolbarItem{Title: "Fit to Window", ID: "fit", Symbol: "arrow.up.left.and.arrow.down.right", Enabled: enabled, OnClick: func() { clicks++ }},
				look.ToolbarItem{Title: "Comment Mode", ID: "comment", Symbol: "text.bubble"})
			look.SetToolbar(w, items...)
			w.SetSize(900, 600)
			w.Show()
			d := app.Unwrap()
			check := func(step, class string) {
				for _, id := range []string{"fit", "comment"} {
					ns, findErr := qatree.Find(d, qatree.Selector{ID: id})
					if findErr != nil || len(ns) != 1 || ns[0].Class != class {
						fail("%s: %s is %+v (%v), want one %s", step, id, ns, findErr, class)
					}
					if id == "fit" && len(ns) == 1 && ns[0].Enabled != enabled.Peek() {
						fail("%s: fit enabled %v, want %v", step, ns[0].Enabled, enabled.Peek())
					}
				}
				bars, _ := qatree.Find(d, qatree.Selector{Class: "ToolBar"})
				for _, bar := range bars {
					if bar.Visible && bar.Bounds[3] != gowt.Spacing.ControlHeight {
						fail("%s: bar height %d, want %d", step, bar.Bounds[3], gowt.Spacing.ControlHeight)
					}
				}
				client := w.Unwrap().GetClientArea()
				buttons, _ := qatree.Find(d, qatree.Selector{Class: "ToolItem"})
				for _, button := range buttons {
					if button.Visible && int32(button.Bounds[0]+button.Bounds[2]) > client.X+client.Width {
						fail("%s: %s ends past the client area: %v", step, button.ID, button.Bounds)
					}
				}
			}
			app.After(500*time.Millisecond, func() {
				check("narrow", "MenuItem")
				if !look.FocusSearch(w) {
					fail("FocusSearch did not find the hidden field")
				}
				search, _ := qatree.Find(d, qatree.Selector{ID: "search"})
				if len(search) != 1 || search[0].Class != "Text" || !search[0].Focused {
					fail("revealed search is %+v, want one focused Text", search)
				}
				enabled.Set(true)
				if actErr := qatree.Act(d, qatree.Selector{ID: "fit"}, "click", json.RawMessage(nil)); actErr != nil {
					fail("click fit: %v", actErr)
				}
				if clicks != 1 {
					fail("fit clicks %d, want 1", clicks)
				}
				w.SetSize(1600, 600)
				app.After(500*time.Millisecond, func() {
					check("wide", "ToolItem")
					w.SetSize(900, 600)
					app.After(500*time.Millisecond, func() {
						check("narrow again", "MenuItem")
						w.Close()
					})
				})
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
