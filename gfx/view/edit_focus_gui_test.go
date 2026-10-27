package view

import (
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

// TestEditFocusGUI types OS key events into Edit fields opened inside a key event: by the view's key
// handler on Space, then by OnCommit, which runs in the Enter that ended the first edit.
func TestEditFocusGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			var v *View
			var ed *Editor
			var commits []string
			edit := func(y float64, onCommit func(string)) {
				ed = v.Edit(EditOptions{Rect: gfx.R(10, y, 160, 30), Font: g.Font{Size: 12}, OnCommit: onCommit})
			}
			next := func(s string) {
				commits = append(commits, s)
				edit(70, func(s string) { commits = append(commits, s) })
			}
			w := app.Window("edit focus", New(gfx.NewScene(), func(x *View) {
				v = x
				v.OnKey(func(k *Key) {
					if k.Down && k.Char == ' ' && ed == nil {
						edit(20, next)
					}
				})
			}, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			w.Unwrap().ForceActive()
			d := app.Unwrap()
			focused := func(phase string) bool {
				if ed == nil {
					t.Errorf("%s: no editor", phase)
					return false
				}
				if c := d.GetFocusControl(); c != &ed.Widget().Unwrap().Control {
					t.Errorf("%s: the focus is on %v, want the editor", phase, c)
				}
				return true
			}
			typed := func(phase, want string) {
				if got := ed.Widget().Text(); got != want {
					t.Errorf("%s: editor text %q, want %q", phase, got, want)
				}
			}
			step := 300 * time.Millisecond
			steps := []func(){
				func() {
					// A window of an earlier run in this process may have left the app inactive.
					w.Unwrap().ForceActive()
					v.Canvas().Unwrap().SetFocus()
				},
				func() {
					if d.GetActiveShell() != w.Unwrap() || !v.Canvas().Unwrap().IsFocusControl() {
						t.Errorf("the window is not active or the view has no focus: the OS did not activate the test")
						return
					}
					postKey(d, ' ')
				},
				func() {
					if focused("after Space") {
						postKey(d, 'x')
					}
				},
				func() {
					typed("first editor", "x")
					postKey(d, swt.CR)
				},
				func() {
					if !slices.Equal(commits, []string{"x"}) {
						t.Errorf("Enter: commits %v, want [x]", commits)
					}
					if focused("after the Enter that opened the second editor") {
						postKey(d, 'y')
					}
				},
				func() { typed("second editor", "y") },
			}
			for i, f := range steps {
				app.After(time.Duration(i+1)*step, func() {
					v.traceFirstResponder(fmt.Sprintf("test step %d begins", i))
					if !t.Failed() {
						f()
					}
				})
			}
			app.After(time.Duration(len(steps)+1)*step, w.Close)
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}

// TestPopoverFromKeyGUI opens a popover with a Focused field from the view's key handler on Space, as
// Tenniarb's operation box does: a key typed by the OS must land in the field.
func TestPopoverFromKeyGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			var v *View
			var field *g.TextView
			var w *g.Window
			w = app.Window("popover from key", New(gfx.NewScene(), func(x *View) {
				v = x
				v.OnKey(func(k *Key) {
					if !k.Down || k.Char != ' ' || field != nil {
						return
					}
					p := w.Popover(g.Gap(4))
					p.Add(g.Text(g.Focused(), g.Ref(&field), g.Hint("Operation")))
					p.ShowAtPoint(v.Canvas(), 100, 100, g.PopoverBelow)
				})
			}, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			w.Unwrap().ForceActive()
			d := app.Unwrap()
			step := 400 * time.Millisecond
			app.After(step, func() {
				w.Unwrap().ForceActive()
				v.Canvas().Unwrap().SetFocus()
			})
			app.After(2*step, func() {
				if d.GetActiveShell() != w.Unwrap() || !v.Canvas().Unwrap().IsFocusControl() {
					t.Errorf("the window is not active or the view has no focus: the OS did not activate the test")
					return
				}
				postKey(d, ' ')
			})
			app.After(4*step, func() {
				if t.Failed() {
					return
				}
				if field == nil {
					t.Error("Space did not reach the view's key handler")
					return
				}
				postKey(d, 'x')
			})
			app.After(5*step, func() {
				defer w.Close()
				if field != nil {
					if got := field.Text(); got != "x" {
						t.Errorf("popover field text %q, want the typed x", got)
					}
				}
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
