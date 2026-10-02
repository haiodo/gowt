package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

// Opens real windows: run only where a display is available (the Linux Xvfb stand).
func TestRun(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var clicks int
	var changed string
	err := Run(func(app *App) {
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
	if err != nil || clicks != 1 || changed != "abc" {
		t.Fatalf("err=%v clicks=%d changed=%q", err, clicks, changed)
	}
}

func TestRunReturnsSWTError(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	err := Run(func(app *App) { swt.Error(swt.ERROR_INVALID_ARGUMENT) })
	if err == nil {
		t.Fatal("want error")
	}
}
