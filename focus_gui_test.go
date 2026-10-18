package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// focusRun builds a window with build, shows it and passes check the control that has the focus right
// after Show and again after the window has had time to be activated.
func focusRun(t *testing.T, build func(app *App, w *Window), check func(when string, focus *swt.Control)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("focus")
			build(app, w)
			w.Show()
			check("after Show", app.display.GetFocusControl())
			app.After(200*time.Millisecond, func() {
				check("after activation", app.display.GetFocusControl())
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func focusName(c *swt.Control) string {
	if c == nil {
		return "nothing"
	}
	if t, ok := c.GetDataKey("name").(string); ok {
		return t
	}
	return nameOf(c)
}

func named(s string) Option { return Custom(func(c *swt.Control) { c.SetDataKeyValue("name", s) }) }

func TestFocused(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []Option
		want string
	}{
		{"text after a button", []Option{Button("first", named("first")), Text(named("text"), Focused())}, "text"},
		{"push button after a button", []Option{Button("first", named("first")), Button("second", named("second"), Focused())}, "second"},
		{"nested, last wins", []Option{Text(named("outer"), Focused()), HBox(Button("b"), Text(named("inner"), Focused()))}, "inner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			focusRun(t, func(app *App, w *Window) { w.Add(tc.opts...) }, func(when string, focus *swt.Control) {
				if got := focusName(focus); got != tc.want {
					t.Errorf("%s: focus on %s, want %s", when, got, tc.want)
				}
			})
		})
	}
}

// A Focused widget added to a window on screen takes the focus on the next turn.
func TestFocusedAfterShow(t *testing.T) {
	added := false
	focusRun(t, func(app *App, w *Window) {
		w.Add(Button("first", named("first")))
		app.Async(func() { w.Add(Text(named("late"), Focused())); added = true })
	}, func(when string, focus *swt.Control) {
		if !added {
			return
		}
		if got := focusName(focus); got != "late" {
			t.Errorf("%s: focus on %s, want late", when, got)
		}
	})
}

func TestFocusedOnWindow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var reported error
	onMain(func() {
		_ = Run(func(app *App) {
			app.OnError(func(err error) { reported = err })
			app.Window("focus", Focused()).Unwrap().Dispose()
		})
	})
	if reported == nil {
		t.Fatal("Focused on a Window was not reported")
	}
}
