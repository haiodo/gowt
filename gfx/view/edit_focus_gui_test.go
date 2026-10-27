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

// whenActive runs start once w has been the active window for half a second: a test process started
// from a terminal on macOS can lose the activation again shortly after launch.
func whenActive(t *testing.T, app *g.App, w *g.Window, start func()) {
	d := app.Unwrap()
	w.Unwrap().ForceActive()
	stable := 0
	var poll func(left int)
	poll = func(left int) {
		if d.GetActiveShell() == w.Unwrap() {
			stable++
		} else {
			stable = 0
			w.Unwrap().ForceActive()
		}
		switch {
		case stable == 5:
			start()
		case left == 0:
			t.Error("the OS did not activate the test window")
			w.Close()
		default:
			app.After(100*time.Millisecond, func() { poll(left - 1) })
		}
	}
	poll(50)
}

type listenerFunc func(*swt.Event)

func (f listenerFunc) HandleEvent(e *swt.Event) { f(e) }

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
			steps := []func(){
				func() {
					if !v.Canvas().Unwrap().SetFocus() {
						t.Error("the view did not take the focus")
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
						t.Errorf("Enter: commits %q, want [x]", commits)
					}
					if focused("after the Enter that opened the second editor") {
						postKey(d, 'y')
					}
				},
				func() { typed("second editor", "y") },
			}
			whenActive(t, app, w, func() {
				// A field ends its edit when the window loses the focus: another app taking it would
				// fail the steps for a reason outside the toolkit.
				done := false
				w.Unwrap().AddListener(swt.Deactivate, listenerFunc(func(*swt.Event) {
					if !done {
						t.Error("another application took the focus during the test")
					}
				}))
				step := 300 * time.Millisecond
				for i, f := range steps {
					app.After(time.Duration(i)*step, func() {
						v.traceFirstResponder(fmt.Sprintf("test step %d begins", i))
						if !t.Failed() {
							f()
						}
					})
				}
				app.After(time.Duration(len(steps))*step, func() {
					done = true
					w.Close()
				})
			})
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
			d := app.Unwrap()
			whenActive(t, app, w, func() {
				step := 400 * time.Millisecond
				if !v.Canvas().Unwrap().SetFocus() {
					t.Error("the view did not take the focus")
					w.Close()
					return
				}
				postKey(d, ' ')
				app.After(2*step, func() {
					if field == nil {
						t.Error("Space did not reach the view's key handler")
						return
					}
					postKey(d, 'x')
				})
				app.After(3*step, func() {
					defer w.Close()
					if field != nil {
						if got := field.Text(); got != "x" {
							t.Errorf("popover field text %q, want the typed x", got)
						}
					}
				})
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
