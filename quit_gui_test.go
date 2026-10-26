package gowt

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

// quitRun runs an app with n shown windows; check runs in a goroutine and ends the app on return.
func quitRun(t *testing.T, n int, check func(app *App, ws []*Window)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	done := make(chan struct{})
	onMain(func() {
		err = Run(func(app *App) {
			var ws []*Window
			for i := 0; i < n; i++ {
				w := app.Window(fmt.Sprint("quit ", i))
				w.Show()
				ws = append(ws, w)
			}
			app.After(30*time.Second, app.Quit) // a stuck check must not hang the run
			go func() {
				defer close(done)
				defer app.Quit()
				check(app, ws)
			}()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	// An allowed quit ends Run while check still reports; t calls must come before the test returns.
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the check did not finish")
	}
}

// quitLog collects the answers of handlers, which run on the UI thread.
type quitLog struct {
	log   []string
	allow map[string]bool
}

func (q *quitLog) ask(name string) func() bool {
	return func() bool {
		q.log = append(q.log, name)
		return q.allow[name]
	}
}

func (q *quitLog) take(app *App) []string {
	return onUI(app, func() []string { l := q.log; q.log = nil; return l })
}

func (q *quitLog) set(app *App, name string, v bool) {
	onUI(app, func() int { q.allow[name] = v; return 0 })
}

func quitAlive(app *App, ws []*Window) int {
	return onUI(app, func() int {
		n := 0
		for _, w := range ws {
			if !w.shell.IsDisposed() {
				n++
			}
		}
		return n
	})
}

func TestQuitAsksWindowsThenHook(t *testing.T) {
	quitRun(t, 2, func(app *App, ws []*Window) {
		q := &quitLog{allow: map[string]bool{"w0": true, "w1": true, "quit": false}}
		onUI(app, func() int {
			ws[0].OnClose(q.ask("w0"))
			ws[1].OnClose(q.ask("w1"))
			app.OnQuit(q.ask("quit"))
			return 0
		})
		closeApp := func() { onUI(app, func() int { app.Unwrap().Close(); return 0 }) }

		closeApp()
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0", "w1", "quit"}) || quitAlive(app, ws) != 2 {
			t.Errorf("quit refused by OnQuit: asked %v, %d windows alive", got, quitAlive(app, ws))
		}
		q.set(app, "w0", false)
		closeApp()
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0"}) || quitAlive(app, ws) != 2 {
			t.Errorf("quit refused by the first window: asked %v, %d windows alive", got, quitAlive(app, ws))
		}
		q.set(app, "w0", true)
		q.set(app, "w1", false)
		closeApp()
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0", "w1"}) || quitAlive(app, ws) != 2 {
			t.Errorf("quit refused by the second window: asked %v, %d windows alive", got, quitAlive(app, ws))
		}
		q.set(app, "w1", true)
		q.set(app, "quit", true)
		var got []string
		var gone bool
		onUI(app, func() int {
			app.Unwrap().Close()
			got, gone = q.log, app.Unwrap().IsDisposed()
			q.log = nil
			return 0
		})
		if !reflect.DeepEqual(got, []string{"w0", "w1", "quit"}) || !gone {
			t.Errorf("quit allowed: asked %v, display disposed = %v", got, gone)
		}
	})
}

func TestQuitLastWindowClose(t *testing.T) {
	quitRun(t, 2, func(app *App, ws []*Window) {
		q := &quitLog{allow: map[string]bool{"w0": true, "w1": true, "quit": false}}
		onUI(app, func() int {
			ws[0].OnClose(q.ask("w0"))
			ws[1].OnClose(q.ask("w1"))
			app.OnQuit(q.ask("quit"))
			ws[0].Close()
			return 0
		})
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0"}) {
			t.Errorf("closing one of two windows asked %v", got)
		}
		onUI(app, func() int { ws[1].Close(); return 0 })
		got := q.take(app)
		if kept := onUI(app, func() bool { return !ws[1].shell.IsDisposed() }); !reflect.DeepEqual(got, []string{"w1", "quit"}) || !kept {
			t.Errorf("closing the last window with OnQuit false: asked %v, window kept = %v", got, kept)
		}
		q.set(app, "quit", true)
		var gone bool
		onUI(app, func() int {
			ws[1].Close()
			got, gone = q.log, ws[1].shell.IsDisposed()
			q.log = nil
			return 0
		})
		if !reflect.DeepEqual(got, []string{"w1", "quit"}) || !gone {
			t.Errorf("closing the last window with OnQuit true: asked %v, window gone = %v", got, gone)
		}
	})
}

// TestQuitDialog shows Confirm from OnQuit, as an editor does, and dismisses it from a timer.
func TestQuitDialog(t *testing.T) {
	quitRun(t, 1, func(app *App, ws []*Window) {
		if !dialogDismissable {
			t.Skip("a native dialog cannot be dismissed in a test on this OS")
		}
		asked := false
		onUI(app, func() int {
			app.OnQuit(func() bool {
				asked = true
				app.After(500*time.Millisecond, func() { dismissDialog(ws[0]) })
				return ws[0].Confirm("Save changes?", "The document has unsaved changes.")
			})
			return 0
		})
		quit := onUI(app, func() bool { app.Unwrap().Close(); return app.Unwrap().IsDisposed() })
		if !asked || quit {
			t.Errorf("dialog in OnQuit: asked=%v, display disposed=%v; a dismissed dialog must cancel", asked, quit)
		}
	})
}

// TestQuitSessionEnd sends what the system sends on logout and shutdown (Windows) or calls
// NSApp terminate: (macOS: Cmd+Q, Dock, logout); other systems skip.
func TestQuitSessionEnd(t *testing.T) {
	quitRun(t, 2, func(app *App, ws []*Window) {
		if !systemQuitAvailable {
			t.Skip("the system quit request is not simulated on this OS")
		}
		q := &quitLog{allow: map[string]bool{"w0": true, "w1": true, "quit": false}}
		onUI(app, func() int {
			ws[0].OnClose(q.ask("w0"))
			ws[1].OnClose(q.ask("w1"))
			app.OnQuit(q.ask("quit"))
			return 0
		})
		if onUI(app, func() bool { return systemQuit(ws[0], false) }) {
			t.Error("the system quit went through while OnQuit said no")
		}
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0", "w1", "quit"}) {
			t.Errorf("system quit asked %v", got)
		}
		q.set(app, "w1", false)
		if onUI(app, func() bool { return systemQuit(ws[0], false) }) {
			t.Error("the system quit went through while a window said no")
		}
		if got := q.take(app); !reflect.DeepEqual(got, []string{"w0", "w1"}) {
			t.Errorf("system quit with a refusing window asked %v", got)
		}
		q.set(app, "w1", true)
		q.set(app, "quit", true)
		if !onUI(app, func() bool { return systemQuit(ws[0], true) }) {
			t.Error("the system quit was refused with every answer yes")
		}
	})
}
