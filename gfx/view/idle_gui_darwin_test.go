//go:build darwin

package view

import (
	"fmt"
	"os"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// TestIdleWindowsGUI is the acceptance check for idle windows: many windows with a static scene
// stop drawing frames and their wantsFrame is false, while a window with an animation (an
// OnFrame driver) keeps ticking at the frame rate.
func TestIdleWindowsGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	const staticN = 20
	var err error
	ok := make(chan struct{})
	mainq <- func() {
		defer close(ok)
		err = g.Run(func(app *g.App) {
			var anim *View
			aw := app.Window("animation", New(gfx.NewScene(), func(v *View) {
				anim = v
				v.Compositor().OnFrame(func(gfx.FrameInfo) {})
			}, g.Grow()))
			aw.SetSize(320, 240)
			aw.Unwrap().SetLocation(int32(20), int32(20))
			aw.Show()

			var views []*View
			var wins []*g.Window
			for i := 0; i < staticN; i++ {
				scene := gfx.NewScene()
				n := gfx.NewNode(1)
				rec := gfx.NewRecorder()
				rec.FillRect(gfx.R(0, 0, 40, 40), gfx.Color{R: 1, A: 1})
				scene.Update(func(tx *gfx.Tx) { tx.Add(nil, n); tx.SetPicture(n, rec.Finish()) })
				w := app.Window(fmt.Sprintf("static-%d", i), New(scene, func(v *View) {
					views = append(views, v)
				}, g.Grow()))
				w.SetSize(200, 150)
				// a grid without overlaps, so no static window occludes another
				w.Unwrap().SetLocation(int32(20+(i%5)*220), int32(320+(i/5)*180))
				w.Show()
				wins = append(wins, w)
			}
			if len(views) != staticN {
				t.Fatalf("%d views, want %d", len(views), staticN)
			}

			// window 0 goes to the Dock, window 1 is fully covered by an opaque window:
			// both must be reported hidden and stop wanting frames
			wins[0].Unwrap().SetMinimized(true)
			cover := app.Window("cover", g.Canvas(nil, g.Grow()))
			cover.SetSize(220, 170)
			cover.Unwrap().SetLocation(int32(20+220), int32(320))
			cover.Show()

			base := make([]uint64, staticN)
			app.After(1500*time.Millisecond, func() {
				for i, v := range views {
					base[i] = v.frames.Load()
				}
			})
			app.After(2300*time.Millisecond, func() {
				for i, v := range views {
					if n := v.frames.Load(); n != base[i] {
						t.Errorf("static window %d drew %d frames while idle", i, n-base[i])
					}
					if v.wantsFrame() {
						t.Errorf("static window %d wants a frame while idle", i)
					}
					if i < 2 && !v.hidden.Load() {
						t.Errorf("static window %d (%s) is not reported hidden", i,
							[2]string{"miniaturized", "occluded"}[i])
					}
				}
				if anim.frames.Load() == 0 {
					t.Error("the animation window drew no frames")
				}
			})
			app.After(3000*time.Millisecond, func() {
				for _, w := range wins {
					w.Close()
				}
				cover.Close()
				aw.Close()
			})
		})
	}
	<-ok
	if err != nil {
		t.Fatal(err)
	}
}
