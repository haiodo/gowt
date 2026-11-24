//go:build darwin

package gowt

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/swt"
)

func TestCanvasDragOutside(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var stage int
	var posted = true
	var skip string
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("drag")
			c := put[*CanvasView](w, Canvas(nil, Size(100, 100), OnMouseDown(func(m Mouse) {
				if m.Button == 1 {
					stage = 1
				}
			}), OnMouseMove(func(m Mouse) {
				if stage == 1 && m.X > 100 {
					stage = 2
				}
			}), OnMouseUp(func(m Mouse) {
				if stage == 2 && m.Button == 1 && m.X > 100 {
					stage = 3
				}
			})))
			w.Show()
			app.After(500*time.Millisecond, func() {
				if skip = postBlocked(app); skip != "" {
					w.Close()
					return
				}
				p := c.Unwrap().ToDisplay(50, 50)
				go func() {
					ok := postMouse(app, swt.MouseMove, p.X, p.Y)
					time.Sleep(100 * time.Millisecond)
					ok = postMouse(app, swt.MouseDown, p.X, p.Y) && ok
					time.Sleep(100 * time.Millisecond)
					ok = postMouse(app, swt.MouseMove, p.X+200, p.Y) && ok
					time.Sleep(100 * time.Millisecond)
					ok = postMouse(app, swt.MouseUp, p.X+200, p.Y) && ok
					time.Sleep(300 * time.Millisecond)
					app.Async(func() { posted = ok; w.Close() })
				}()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if skip != "" {
		t.Skip(skip)
	}
	if !posted {
		t.Fatal("failed to post mouse event")
	}
	if stage != 3 {
		t.Fatalf("drag reached stage %d, want 3", stage)
	}
}
