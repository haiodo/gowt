package view

import (
	"os"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// directGUIScene is a background with one shadowed card, like the tenniarb document the Direct
// mode is for.
func directGUIScene() *gfx.Scene {
	scene := gfx.NewScene()
	bg := gfx.NewNode(1)
	bgr := gfx.NewRecorder()
	bgr.FillRect(gfx.R(-1000, -1000, 4000, 3000), gfx.RGB(255, 255, 255))
	card := gfx.NewNode(2)
	rec := gfx.NewRecorder()
	rec.FillShadowRRect(0, 0, 300, 180, 12, gfx.RGB(30, 120, 220), gfx.Shadow{DX: 0, DY: 8, Blur: 16, Color: gfx.Color{A: 0.4}})
	rec.Text("Direct", 20, 60, gfx.Font{Size: 32}, gfx.White)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, bg)
		tx.SetPicture(bg, bgr.Finish())
		tx.Add(nil, card)
		tx.SetPicture(card, rec.Finish())
		tx.Move(card, 120, 100)
	})
	return scene
}

// TestDirectGUI runs the Direct mode in a real window on every OS: the camera sweeps the zooms
// of the task's acceptance, the view resizes, and the HUD scene is served. Assertions avoid
// counting display frames: an occluded or background test window paints nothing by design, and
// frame-driven behavior is covered deterministically in gfx and by the HUD/idle GUI tests. The
// pixel-level proof that Direct draws at exactly zoom*dpr with an unweakened shadow is the
// headless gfx/cg TestDirectExactScalePixels; a live two-window screenshot comparison was
// attempted and removed as unreliable, see .work/direct-scene-draw/agent-32/deviations.md.
func TestDirectGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			scene := directGUIScene()
			var v *View
			var framesAtResize uint64
			w := app.Window("direct", NewWithOptions(scene, Options{Direct: true}, func(x *View) {
				v = x
				hud := x.Compositor().HUD()
				badge := gfx.NewNode(1)
				rec := gfx.NewRecorder()
				rec.FillRect(gfx.R(0, 0, 24, 12), gfx.Color{R: 1, A: 1})
				hud.Update(func(tx *gfx.Tx) { tx.Add(nil, badge); tx.SetPicture(badge, rec.Finish()); tx.Move(badge, 8, 8) })
			}, g.Grow()))
			w.SetSize(700, 500)
			w.Show()

			app.After(300*time.Millisecond, func() {
				if got := v.Compositor().Options(); !got.Direct {
					t.Errorf("compositor options %+v, want Direct", got)
				}
				if v.Compositor().HUD() == nil {
					t.Error("no HUD scene")
				}
			})
			// the acceptance sweep: 1x -> 1.3x -> 2x -> 3x -> 4x
			for i, z := range []float64{1, 1.3, 2, 3, 4} {
				z := z
				app.After(time.Duration(400+300*i)*time.Millisecond, func() {
					v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: 0, Y: 0}, Zoom: z})
				})
			}
			app.After(2*time.Second, func() {
				if z := v.Compositor().Camera().Zoom; z != 4 {
					t.Errorf("zoom %v, want 4 kept", z)
				}
				framesAtResize = v.frames.Load()
				w.SetSize(500, 400) // resize: the viewport follows and the view redraws
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: 0, Y: 0}, Zoom: 1.3})
			})
			app.After(3*time.Second, func() {
				if z := v.Compositor().Camera().Zoom; z != 1.3 {
					t.Errorf("zoom %v after the resize, want 1.3 kept", z)
				}
				if v.frames.Load() > framesAtResize {
					vw, vh, _ := v.Compositor().Viewport()
					if vw >= 700 || vh >= 500 {
						t.Errorf("viewport %vx%v after SetSize(500, 400), want it smaller", vw, vh)
					}
				} else {
					t.Log("the window painted no frame after the resize (hidden or occluded): the viewport follow is not checked")
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
