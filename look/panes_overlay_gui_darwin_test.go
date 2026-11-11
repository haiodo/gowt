package look_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	viewpkg "github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// With panes overlay the content runs the full client area under the panes; with a gfx canvas in
// the content its CALayer used to composite above the sidebar and inspector whatever the NSView
// order, and the panes were invisible on screen (see gowt-task-panes-overlay). The capture must
// show the red markers of the panes over the green canvas, and the canvas view must stay the
// lowest of the three in the window's subview order, also after hiding and showing a pane.
func TestPanesOverlayGfxZOrder(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	overlayOn := runOverlayApp(t, true, func(w *gowt.Window, side, content, insp *gowt.Panel) {
		if !cocoa.WantsLayer(w.Unwrap().View.Id) {
			t.Error("overlay left the window content view without a backing layer")
		}
		panes := map[string]*gowt.Panel{"side": side, "content": content, "insp": insp}
		checkPaneOrder(t, paneOrder(content, side, insp), "shown")
		img, width := capturePNG(t, w, "overlay.png")
		checkPaneColors(t, img, paneRects(w, panes), width)
		look.SetSidebarVisible(w, false)
		look.SetSidebarVisible(w, true)
		checkPaneOrder(t, paneOrder(content, side, insp), "after hide/show")
	})
	if !overlayOn {
		t.Skip("panes overlay needs the macOS 26 glass look")
	}
}

// SetPanesOverlay in the other order, after SetSidebar and SetInspector, must layer-back the
// window content view as well; turning it off restores backing-store rendering and the side by
// side layout, turning it on again restores the overlay.
func TestPanesOverlayToggleAndOrder(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	overlayOn := runOverlayApp(t, false, func(w *gowt.Window, side, content, insp *gowt.Panel) {
		panes := map[string]*gowt.Panel{"side": side, "content": content, "insp": insp}
		if !cocoa.WantsLayer(w.Unwrap().View.Id) {
			t.Error("overlay after SetSidebar/SetInspector left the content view without a backing layer")
			return
		}
		checkPaneOrder(t, paneOrder(content, side, insp), "overlay last")
		look.SetPanesOverlay(w, false)
		if cocoa.WantsLayer(w.Unwrap().View.Id) {
			t.Error("SetPanesOverlay(false) left the content view layer-backed")
		}
		img, width := capturePNG(t, w, "overlay-off.png")
		checkPaneColors(t, img, paneRects(w, panes), width)
		look.SetPanesOverlay(w, true)
		if !cocoa.WantsLayer(w.Unwrap().View.Id) {
			t.Error("SetPanesOverlay(true) again left the content view without a backing layer")
		}
		checkPaneOrder(t, paneOrder(content, side, insp), "overlay again")
		img, width = capturePNG(t, w, "overlay-on-again.png")
		checkPaneColors(t, img, paneRects(w, panes), width)
	})
	if !overlayOn {
		t.Skip("panes overlay needs the macOS 26 glass look")
	}
}

// runOverlayApp builds the overlay test window - SetPanesOverlay before the panes where overlayFirst,
// after them where not - and runs act on the UI thread once the window showed. It reports whether
// the overlay engaged (it does not without the macOS 26 glass).
func runOverlayApp(t *testing.T, overlayFirst bool, act func(*gowt.Window, *gowt.Panel, *gowt.Panel, *gowt.Panel)) bool {
	t.Helper()
	var overlayOn bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("overlay z-order")
			var side, content, insp *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&side), gowt.Canvas(fillRed, gowt.Grow())),
				gowt.VBox(gowt.Ref(&content), viewpkg.New(greenScene(), nil, gowt.Grow())),
				gowt.VBox(gowt.Ref(&insp), gowt.Canvas(fillRed, gowt.Grow())))
			if overlayFirst {
				look.SetPanesOverlay(w, true)
			}
			look.SetSidebar(w, side, content, 220)
			look.SetInspector(w, content, insp, 260)
			if !overlayFirst {
				look.SetPanesOverlay(w, true)
			}
			w.SetSize(900, 600)
			w.Show()
			app.After(time.Second, func() {
				defer w.Close()
				overlayOn, _ = w.Unwrap().GetDataKey("look.panesOverlay").(bool)
				act(w, side, content, insp)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return overlayOn
}

// greenScene is a solid green canvas, large enough to fill the viewport at any pan and zoom.
func greenScene() *gfx.Scene {
	scene := gfx.NewScene()
	scene.Update(func(tx *gfx.Tx) {
		rec := gfx.NewRecorder()
		rec.FillRect(gfx.R(0, 0, 8000, 8000), gfx.RGB(30, 200, 60))
		n := gfx.NewNode(1)
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})
	return scene
}

func fillRed(g *gowt.GC) {
	w, h := g.Size()
	g.SetFill(gowt.RGB{R: 220, G: 40, B: 40})
	g.FillRect(0, 0, w, h)
}

// paneRects is a horizontal strip at the middle of each pane, in window points from the window's
// top left corner.
func paneRects(w *gowt.Window, panes map[string]*gowt.Panel) map[string][4]int32 {
	s := w.Unwrap().GetBounds()
	rects := map[string][4]int32{}
	for name, p := range panes {
		o, size := p.Unwrap().ToDisplay(0, 0), p.Unwrap().GetSize()
		rects[name] = [4]int32{o.X - s.X + size.X/4, o.Y - s.Y + size.Y/2, size.X / 2, 8}
	}
	return rects
}

// paneOrder is the subview index of the content, sidebar and inspector in the window's content
// view: a higher index draws above a lower one.
func paneOrder(content, side, insp *gowt.Panel) []int {
	host := content.Unwrap().View.Superview()
	idx := make([]int, 3)
	for i, p := range []*gowt.Panel{content, side, insp} {
		idx[i] = cocoa.SubviewIndex(host.Id, p.Unwrap().View.Id)
	}
	return idx
}

func checkPaneOrder(t *testing.T, idx []int, when string) {
	t.Helper()
	for i, v := range idx {
		if v < 0 {
			t.Fatalf("%s: pane %d not a subview of the window content view", when, i)
		}
	}
	if idx[0] > idx[1] || idx[0] > idx[2] {
		t.Errorf("%s: content view above a pane in subview order %v", when, idx)
	}
}

// capturePNG screenshots the window and decodes the PNG; the second value is the window width in
// points, for scaling window points to pixels.
func capturePNG(t *testing.T, w *gowt.Window, name string) (image.Image, int32) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if dir := os.Getenv("GOWT_PIXELS_DIR"); dir != "" {
		path = filepath.Join(dir, name)
	}
	if err := shot.WindowPNG(w.Unwrap().View.Id, path); err != nil {
		t.Error(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), 1
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), 1
	}
	return img, w.Unwrap().GetBounds().Width
}

// checkPaneColors asserts the strip of each pane: the content shows the green canvas, the panes
// their red marker; the canvas covering a pane is what the bug looked like.
func checkPaneColors(t *testing.T, img image.Image, rects map[string][4]int32, width int32) {
	t.Helper()
	scale := float64(img.Bounds().Dx()) / float64(width)
	for name, r := range rects {
		m := meanRGB(img, r, scale)
		rr, gg, bb := m[0], m[1], m[2]
		t.Logf("%s at %v: r %.0f g %.0f b %.0f", name, r, rr, gg, bb)
		switch name {
		case "content":
			if gg < rr+40 || gg < bb+40 {
				t.Errorf("content at %v is not the green canvas: r %.0f g %.0f b %.0f", r, rr, gg, bb)
			}
		default:
			if rr < gg+40 || rr < bb+40 {
				t.Errorf("%s at %v shows the canvas through, not the pane: r %.0f g %.0f b %.0f", name, r, rr, gg, bb)
			}
		}
	}
}
