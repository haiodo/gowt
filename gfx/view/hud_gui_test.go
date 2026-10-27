package view

import (
	"os"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// TestHUDGUI runs a HUD over a real view: a badge pinned to the right edge keeps its place while
// the camera pans and zooms, follows the edge when the window is resized, and answers the HUD hit test.
func TestHUDGUI(t *testing.T) {
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
			var badge *gfx.Node
			w := app.Window("hud", New(scene, func(x *View) {
				v = x
				hud := v.Compositor().HUD()
				badge = gfx.NewNode(1)
				rec := gfx.NewRecorder()
				rec.FillRect(gfx.R(0, 0, 40, 20), gfx.Color{R: 1, A: 1})
				hud.Update(func(tx *gfx.Tx) { tx.Add(nil, badge); tx.SetPicture(badge, rec.Finish()) })
				v.Compositor().OnViewport(func(vw, vh, dpr float64) {
					hud.Update(func(tx *gfx.Tx) { tx.Move(badge, vw-40, 10) })
				})
			}, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			step := 300 * time.Millisecond
			at := func() gfx.Point {
				m := badge.World()
				return gfx.Point{X: m[4], Y: m[5]}
			}
			var width float64
			app.After(step, func() {
				width, _, _ = v.Compositor().Viewport()
				if p := at(); p.X != width-40 || p.Y != 10 {
					t.Errorf("badge at %v, want the right edge of %v", p, width)
				}
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: 300, Y: 200}, Zoom: 3})
			})
			app.After(2*step, func() {
				if p := at(); p.X != width-40 || p.Y != 10 {
					t.Errorf("the camera moved the badge: %v", p)
				}
				w.SetSize(500, 300)
			})
			app.After(3*step, func() {
				nw, _, _ := v.Compositor().Viewport()
				if nw <= width {
					t.Errorf("the view did not grow: %v -> %v", width, nw)
				}
				if p := at(); p.X != nw-40 {
					t.Errorf("after the resize badge at %v, want x %v", p, nw-40)
				}
				if n := v.HUDHitTest(int(nw)-20, 20); n != badge {
					t.Errorf("hit test over the badge: %v", n)
				}
				if n := v.HUDHitTest(10, 10); n != nil {
					t.Errorf("hit test beside the badge: %v", n)
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
