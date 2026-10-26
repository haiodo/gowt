package view

import (
	"os"
	"runtime"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

var mainq = make(chan func())

func init() { runtime.LockOSThread() }

// The toolkit wants one thread: the tests run on their own goroutines and hand work to this one.
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

// postWheel moves the pointer to the point of c at num/4 of its size and sends one wheel notch through SendInput.
func postWheel(c *swt.Control, num int32) {
	b := c.GetBounds()
	p := c.GetParent().ToDisplay(b.X+b.Width*num/4, b.Y+b.Height*num/4)
	d := c.GetDisplay()
	move := swt.NewEvent()
	move.Type, move.X, move.Y = swt.MouseMove, p.X, p.Y
	d.Post(move)
	time.Sleep(50 * time.Millisecond)
	wheel := swt.NewEvent()
	wheel.Type, wheel.Detail, wheel.Count = swt.MouseWheel, swt.SCROLL_LINE, -3
	d.Post(wheel)
}

func postKey(d *swt.Display, char uint16) {
	e := swt.NewEvent()
	e.Character = char
	e.Type = swt.KeyDown
	d.Post(e)
	e.Type = swt.KeyUp
	d.Post(e)
}

// TestOverlayEditGUI runs an Edit over a real view: a child window over the Direct2D swap chain
// follows the camera, takes the wheel over it, and Esc and Enter end the edit.
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
			w.Unwrap().ForceActive()
			var commits, cancels []string
			var ed *Editor
			step := 300 * time.Millisecond
			rect := gfx.R(10, 20, 100, 30)
			startEdit := func(text string) {
				ed = v.Edit(EditOptions{Rect: rect, Text: text,
					OnCommit: func(s string) { commits = append(commits, s) }, OnCancel: func() { cancels = append(cancels, "x") }})
			}
			app.After(step, func() {
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: -40, Y: -40}, Zoom: 1.5})
				startEdit("hello")
			})
			var before gfx.Camera
			app.After(2*step, func() {
				cam := gfx.Camera{Origin: gfx.Point{X: -40, Y: -40}, Zoom: 1.5}
				x, y, wd, h := childRect(cam, rect)
				b := ed.Widget().Unwrap().GetBounds()
				if int(b.X) != x || int(b.Y) != y || int(b.Width) != wd || int(b.Height) != h {
					t.Errorf("bounds %v, want %d,%d %dx%d", b, x, y, wd, h)
				}
				before = v.Compositor().Camera()
				postWheel(&ed.Widget().Unwrap().Control, 2)
			})
			app.After(3*step, func() {
				if cam := v.Compositor().Camera(); cam != before {
					t.Errorf("wheel over the field moved the camera: %v -> %v", before, cam)
				}
				postWheel(&v.Canvas().Unwrap().Control, 3)
			})
			app.After(4*step, func() {
				if cam := v.Compositor().Camera(); cam == before {
					t.Errorf("wheel over the view did not move the camera: %v", cam)
				}
				ed.Widget().Unwrap().SetFocus()
				postKey(app.Unwrap(), swt.CR)
			})
			app.After(5*step, func() {
				if len(commits) != 1 || commits[0] != "hello" || len(cancels) != 0 || len(v.overlays) != 0 {
					t.Errorf("Enter: commits %v cancels %v overlays %d", commits, cancels, len(v.overlays))
				}
				startEdit("again")
			})
			app.After(6*step, func() {
				ed.Widget().Unwrap().SetFocus()
				postKey(app.Unwrap(), swt.ESC)
			})
			app.After(7*step, func() {
				if len(commits) != 1 || len(cancels) != 1 || len(v.overlays) != 0 {
					t.Errorf("Esc: commits %v cancels %v overlays %d", commits, cancels, len(v.overlays))
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
