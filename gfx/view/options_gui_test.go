package view

import (
	"os"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// TestOptionsGUI runs a real view created with NewWithOptions on every OS and checks that the
// options reached the compositor, that a plain New keeps the compositor defaults, and that the
// view still pans, zooms and hosts a HUD with non-default options.
func TestOptionsGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			want := Options{TilePx: 256, Workers: 2, Margin: 3, Settle: 50 * time.Millisecond, ExactZoom: true}
			scene := gfx.NewScene()
			var v *View
			var def *View
			var hudNode *gfx.Node
			w := app.Window("options", NewWithOptions(scene, want, func(x *View) {
				v = x
				hud := x.Compositor().HUD()
				hudNode = gfx.NewNode(1)
				rec := gfx.NewRecorder()
				rec.FillRect(gfx.R(0, 0, 20, 10), gfx.Color{R: 1, A: 1})
				hud.Update(func(tx *gfx.Tx) { tx.Add(nil, hudNode); tx.SetPicture(hudNode, rec.Finish()); tx.Move(hudNode, 5, 6) })
			}, g.Grow()))
			w.Add(New(gfx.NewScene(), func(x *View) { def = x }, g.Grow()))
			w.SetSize(400, 300)
			w.Show()
			app.After(300*time.Millisecond, func() {
				if got := v.Compositor().Options(); got != gfx.Options(want) {
					t.Errorf("compositor options = %+v, want %+v", got, want)
				}
				d := def.Compositor().Options()
				if d.TilePx != 512 || d.Margin != 1 || d.Settle != 120*time.Millisecond || d.ExactZoom {
					t.Errorf("default view compositor options = %+v", d)
				}
				v.Compositor().SetCamera(gfx.Camera{Origin: gfx.Point{X: 10, Y: 20}, Zoom: 1.3})
			})
			app.After(600*time.Millisecond, func() {
				if p := hudNode.World(); p[4] == 0 && p[5] == 0 {
					t.Error("HUD node did not draw")
				}
				if z := v.Compositor().Camera().Zoom; z != 1.3 {
					t.Errorf("zoom = %v, want 1.3 kept with ExactZoom", z)
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
