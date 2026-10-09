package icons

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
)

var mainq = make(chan func())

// Tests run on their own goroutines; AppKit needs the main thread, which gowt's init pinned to this one.
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

func TestIconsRender(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("t")
			w.SetLayout(gowt.Grid{Columns: 8, Margin: 5})
			for _, n := range Names() {
				img := Get(app, n, Size(20))
				if wd, h := img.Size(); wd != 20 || h != 20 {
					t.Errorf("%s: size %dx%d", n, wd, h)
				}
				w.Button("", nil, gowt.Tooltip(n)).SetImage(img)
			}
			w.Show()
			app.After(10*time.Millisecond, w.Close)
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
