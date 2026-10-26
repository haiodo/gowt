package view

import (
	"os"
	"runtime"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/gtk"
)

var mainq = make(chan func())

func init() { runtime.LockOSThread() }

// GTK wants every call on one thread: the tests run on their own goroutines and hand work to this one.
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

// TestOverlayEditGUI runs an Edit over a real view under Xvfb: the control follows the camera,
// takes the wheel over it, and a press on the view ends the edit once.
func TestOverlayEditGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			scene := gfx.NewScene()
			var v *View
			w := app.Window("overlay", New(scene, func(x *View) { v = x }, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			var commits, cancels []string
			var ed *Editor
			step := 250 * time.Millisecond
			rect := gfx.R(10, 20, 100, 30)
			app.After(step, func() {
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: -40, Y: -40}, Zoom: 1.5})
				ed = v.Edit(EditOptions{Rect: rect, Text: "hello", Font: g.Font{Size: 12},
					OnCommit: func(s string) { commits = append(commits, s) }, OnCancel: func() { cancels = append(cancels, "x") }})
			})
			want := func(phase string, cam gfx.Camera) {
				x, y, wd, h := childRect(cam, rect)
				b := ed.Widget().Unwrap().GetBounds()
				if int(b.X) != x || int(b.Y) != y || int(b.Width) != wd || int(b.Height) != h {
					t.Errorf("%s: bounds %v, want %d,%d %dx%d", phase, b, x, y, wd, h)
				}
			}
			app.After(2*step, func() {
				want("first camera", gfx.Camera{Origin: gfx.Point{X: -40, Y: -40}, Zoom: 1.5})
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: 5, Y: 0}, Zoom: 3})
			})
			var before gfx.Camera
			app.After(3*step, func() {
				want("pan and zoom", gfx.Camera{Origin: gfx.Point{X: 5, Y: 0}, Zoom: 3})
				before = v.Compositor().Camera()
				gtk.PostWheel(ed.Widget().Unwrap().Handle, true)
			})
			app.After(4*step, func() {
				if cam := v.Compositor().Camera(); cam != before {
					t.Errorf("wheel over the field moved the camera: %v -> %v", before, cam)
				}
				gtk.PostWheel(v.Canvas().Unwrap().Handle, true)
			})
			app.After(5*step, func() {
				if cam := v.Compositor().Camera(); cam == before {
					t.Errorf("wheel over the view did not move the camera: %v", cam)
				}
				ed.Commit()
				ed.Commit()
			})
			app.After(6*step, func() {
				if len(commits) != 1 || commits[0] != "hello" || len(cancels) != 0 {
					t.Errorf("commit: commits %v cancels %v", commits, cancels)
				}
				if len(v.overlays) != 0 {
					t.Errorf("%d overlays after the commit", len(v.overlays))
				}
				ed = v.Edit(EditOptions{Rect: rect, Text: "again", OnCommit: func(s string) { commits = append(commits, s) }})
			})
			app.After(7*step, func() {
				// a press on the view commits
				if len(v.overlays) != 1 {
					t.Errorf("%d overlays while editing", len(v.overlays))
				}
				v.r.pointer(PointerDown, gfx.Point{X: 300, Y: 250}, 1, 1, g.Mods{}) // a press on the view
			})
			app.After(8*step, func() {
				if len(v.overlays) != 0 {
					t.Errorf("%d overlays after the press", len(v.overlays))
				}
				if len(commits) != 2 || commits[1] != "again" {
					t.Errorf("a press on the view: commits %v", commits)
				}
				w.Close()
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
