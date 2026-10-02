package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

var mainq = make(chan func())

// Tests run on their own goroutines; AppKit needs the main thread, which init pinned to this one.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func onMain(f func()) {
	ok := make(chan struct{})
	mainq <- func() { defer close(ok); f() }
	<-ok
}

// Opens real windows: run only where a display is available.
func TestRun(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var clicks int
	var changed string
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("t")
			w.SetLayout(Grid{Columns: 2, Margin: 5})
			txt := w.Text(Cell(GridCell{GrowX: true, Align: AlignFill}))
			txt.OnChange(func(s string) { changed = s })
			btn := w.Button("go", func() { clicks++ })
			w.Show()
			txt.SetText("abc")
			btn.Unwrap().NotifyListeners(swt.Selection, swt.NewEvent())
			app.After(10*time.Millisecond, func() { w.Close() })
		})
	})
	if err != nil || clicks != 1 || changed != "abc" {
		t.Fatalf("err=%v clicks=%d changed=%q", err, clicks, changed)
	}
}

func TestRunReturnsSWTError(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() { err = Run(func(app *App) { swt.Error(swt.ERROR_INVALID_ARGUMENT) }) })
	if err == nil {
		t.Fatal("want error")
	}
}
