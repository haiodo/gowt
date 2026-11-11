package look_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	viewpkg "github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
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

// With panes overlay the bottom panel floats over the full-height content, inset from the
// column's sides and the foot like the side panes: it keeps its bounds, sash, hide/show and
// height; the content keeps the whole client area. Off restores the old numbers.
func TestPanesOverlayBottomPanelFloats(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	overlayOn := runOverlayBottomApp(t, nil, func(app *gowt.App, w *gowt.Window, side, content, insp, bottom *gowt.Panel) {
		shell := w.Unwrap()
		r := shell.GetClientArea()
		full := [4]int32{r.X, r.Y, r.Width, r.Height}
		if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
			t.Errorf("overlay content %v, want the whole client area %v", cb, full)
		}
		pb := bottom.Unwrap().GetBounds()
		// The panel floats in the central column: inset from the sidebar's and the inspector's
		// inner edges and from the foot, not under either side pane.
		sb, ib := side.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
		in := sb.X - r.X // the floating glass inset, one on every side of the panel
		want := [4]int32{sb.X + sb.Width + in, r.Y + r.Height - 160 - in, ib.X - sb.X - sb.Width - 2*in, 160}
		if [4]int32{pb.X, pb.Y, pb.Width, pb.Height} != want {
			t.Errorf("bottom panel %v, want %v: floating in the central column", pb, want)
		}
		// The side panes run the full client height; their children stop a glass inset above
		// the foot, off the glass's rounded bottom corners.
		if sb.Y+sb.Height != r.Y+r.Height-2*in {
			t.Errorf("sidebar %v, want its children off the rounded corners %d above the foot", sb, 2*in)
		}
		if ib.Y+ib.Height != r.Y+r.Height-2*in {
			t.Errorf("inspector %v, want its children off the rounded corners %d above the foot", ib, 2*in)
		}
		// All three panes float with the same rounded glass; flush would be 0.
		for name, p := range map[string]*gowt.Panel{"sidebar": side, "inspector": insp, "bottom": bottom} {
			rad := cocoa.GlassCornerRadius(p.AsComposite().View)
			t.Logf("radius %s: %v", name, rad)
			if rad <= 0 {
				t.Errorf("%s glass corner radius %v, want rounded like the side panes", name, rad)
			}
		}
		sash := findBottomSash(t, shell, bottom)
		if sh := sash.GetBounds(); sh.Y != pb.Y || sh.Width != pb.Width {
			t.Errorf("sash %v, want on the panel's top edge %v", sh, pb)
		}

		look.SetBottomPanelHeight(w, 60)
		app.Flush()
		if h := look.BottomPanelHeight(w); h != 60 {
			t.Errorf("SetBottomPanelHeight(60) left the height %d", h)
		}
		if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
			t.Errorf("content after a height change %v, want still the whole client area %v", cb, full)
		}
		if pb := bottom.Unwrap().GetBounds(); pb.Height != 60 || pb.Y+pb.Height != r.Y+r.Height-in {
			t.Errorf("panel after SetBottomPanelHeight(60) %v, want 60 high %d above the foot", pb, in)
		}

		// The sash drag: two Selection events 30 points up raise the panel by 30; the release
		// tells OnBottomPanelChange.
		var changed [2]int
		look.OnBottomPanelChange(w, func(visible bool, height int) { changed[0], changed[1] = b2i(visible), height })
		sy := sash.GetBounds().Y
		postSash(sash, swt.MouseDown, sy)
		postSash(sash, swt.Selection, sy-10)
		postSash(sash, swt.Selection, sy-30)
		postSash(sash, swt.MouseUp, sy-30)
		if h := look.BottomPanelHeight(w); h != 90 {
			t.Errorf("dragging the sash 30 points up left the height %d, want 90", h)
		}
		if changed[0] != 1 || changed[1] != 90 {
			t.Errorf("OnBottomPanelChange after the drag got visible=%d height=%d, want 1 90", changed[0], changed[1])
		}
		if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
			t.Errorf("content after the sash drag %v, want still the whole client area %v", cb, full)
		}

		look.SetBottomPanelVisible(w, false)
		app.Flush()
		if f, _, _ := cocoa.BackdropFrame(bottom.AsComposite().View, nil); f.Height != 0 {
			t.Errorf("panel hidden: glass %v, want 0 high", f)
		}
		if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
			t.Errorf("content with the panel hidden %v, want still the whole client area %v", cb, full)
		}
		look.SetBottomPanelVisible(w, true)
		app.Flush()

		// Flush: the panel fills the central column edge to edge and runs to the foot, square;
		// floating again restores the inset glass.
		flushWant := [4]int32{sb.X + sb.Width, r.Y + r.Height - 90, ib.X - sb.X - sb.Width, 90}
		look.SetBottomPanelFloating(w, false)
		app.Flush()
		if pb := bottom.Unwrap().GetBounds(); [4]int32{pb.X, pb.Y, pb.Width, pb.Height} != flushWant {
			t.Errorf("flush bottom %v, want %v: the column edge to edge, to the foot", pb, flushWant)
		}
		if rad := cocoa.GlassCornerRadius(bottom.AsComposite().View); rad != 0 {
			t.Errorf("flush bottom glass corner radius %v, want 0", rad)
		}
		look.SetBottomPanelFloating(w, true)
		app.Flush()
		floatWant := [4]int32{sb.X + sb.Width + in, r.Y + r.Height - 90 - in, ib.X - sb.X - sb.Width - 2*in, 90}
		if pb := bottom.Unwrap().GetBounds(); [4]int32{pb.X, pb.Y, pb.Width, pb.Height} != floatWant {
			t.Errorf("floating again bottom %v, want %v", pb, floatWant)
		}

		// Overlay off: the content drops below the toolbar and the panel again, in the column
		// between the sidebar and the inspector: the old numbers.
		look.SetPanesOverlay(w, false)
		app.Flush()
		top := int32(cocoa.ContentTopInset(shell.View))
		sb, ib = side.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
		cb := content.Unwrap().GetBounds()
		wantOff := [4]int32{sb.X + sb.Width, r.Y + top, ib.X - sb.X - sb.Width, max(r.Height-top-90, 0)}
		if [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != wantOff {
			t.Errorf("overlay off content %v, want %v", cb, wantOff)
		}
		look.SetPanesOverlay(w, true)
		app.Flush()
		if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
			t.Errorf("overlay on again content %v, want %v", cb, full)
		}
	})
	if !overlayOn {
		t.Skip("panes overlay needs the macOS 26 glass look")
	}
}

// The bottom panel and the native toolbar draw over the gfx canvas: the capture shows the scene
// under both, and clicks reach the panel and the toolbar, not the canvas beneath.
func TestPanesOverlayBottomAndToolbarGfx(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var reduce bool
	onMain(func() { reduce = cocoa.ReduceTransparency() })
	counters := &clickCounters{}
	overlayOn := runOverlayBottomApp(t, counters, func(app *gowt.App, w *gowt.Window, side, content, insp, bottom *gowt.Panel) {
		shell := w.Unwrap()
		r := shell.GetClientArea()
		host := content.Unwrap().View.Superview()
		if ci, bi := cocoa.SubviewIndex(host.Id, content.Unwrap().View.Id), cocoa.SubviewIndex(host.Id, bottom.Unwrap().View.Id); bi < 0 || ci < 0 || bi < ci {
			t.Errorf("bottom panel %d not above the content %d in subview order", bi, ci)
		}
		if reduce {
			t.Log("Reduce transparency: the pixel checks are skipped; the fallback keeps opaque panels")
		} else {
			panes := map[string][4]int32{
				"content": {r.X + r.Width/2 - 20, r.Y + r.Height/2, 40, 8},
				"bottom":  {r.X + r.Width/2 - 20, r.Y + r.Height - 30, 40, 8},
				"toolbar": {r.X + r.Width/2 - 20, r.Y + 6, 40, 8},
			}
			img, width := capturePNG(t, w, "overlay-bottom.png")
			checkPaneColors(t, img, panes, width)
		}
		if tb, ok := qaview.ToolbarOf(shell); ok && len(tb.Buttons()) > 0 {
			b := tb.Buttons()[0].Bounds
			postWindowClick(content.Unwrap(), int32(b[0]+b[2]/2), int32(b[1]+b[3]/2))
		}
		postWindowClick(content.Unwrap(), r.X+r.Width-30, 8) // the toolbar's blank area over the inspector
		pb := bottom.Unwrap().GetBounds()
		if counters.hit != nil {
			hb := counters.hit.Unwrap().GetBounds()
			postWindowClick(content.Unwrap(), pb.X+hb.X+hb.Width/2, pb.Y+hb.Y+hb.Height/2)
		}
		postWindowClick(content.Unwrap(), pb.X+pb.Width/2, pb.Y+pb.Height-12) // the panel's blank glass
		postWindowClick(content.Unwrap(), r.X+r.Width/2, r.Y+r.Height/2)      // the content itself
		app.After(500*time.Millisecond, func() {
			if counters.toolbar != 1 {
				t.Errorf("clicking the toolbar item reached it %d times, want 1", counters.toolbar)
			}
			if counters.hitClicks != 1 {
				t.Errorf("clicking the panel button reached it %d times, want 1", counters.hitClicks)
			}
			if counters.content != 1 {
				t.Errorf("clicks on the panel and toolbar leaked to the content %d times, want 1 (the content click only)", counters.content)
			}
		})
	})
	if !overlayOn {
		t.Skip("panes overlay needs the macOS 26 glass look")
	}
}

// Both side-pane looks, floating and flush, switch live: floating insets the panes from the
// window edges, full-height runs their glass to the top and bottom edges with no corners; the
// bottom panel stays the central column between them and the content keeps the whole client.
func TestPanesOverlayFullHeightPanes(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	overlayOn := runOverlayBottomApp(t, nil, func(app *gowt.App, w *gowt.Window, side, content, insp, bottom *gowt.Panel) {
		r := w.Unwrap().GetClientArea()
		full := [4]int32{r.X, r.Y, r.Width, r.Height}
		var in int32 // the side panes' floating glass inset
		check := func(when string, bin int32) {
			t.Helper()
			sb, ib := side.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
			pb := bottom.Unwrap().GetBounds()
			if cb := content.Unwrap().GetBounds(); [4]int32{cb.X, cb.Y, cb.Width, cb.Height} != full {
				t.Errorf("%s: content %v, want the whole client area %v", when, cb, full)
			}
			if pb.X != sb.X+sb.Width+bin || pb.X+pb.Width != ib.X-bin || pb.Y+pb.Height != r.Y+r.Height-bin {
				t.Errorf("%s: bottom %v, want floating between the panes, %d off the foot", when, pb, bin)
			}
			// The panes' glass rounds all corners; flush is square.
			for name, p := range map[string]*gowt.Panel{"sidebar": side, "inspector": insp} {
				if rad := cocoa.GlassCornerRadius(p.AsComposite().View); (rad > 0) != (in > 0) {
					t.Errorf("%s: %s glass corner radius %v, floating %v", when, name, rad, in > 0)
				}
			}
			if rad := cocoa.GlassCornerRadius(bottom.AsComposite().View); (rad > 0) != (bin > 0) {
				t.Errorf("%s: bottom glass corner radius %v, floating %v", when, rad, bin > 0)
			}
			if in > 0 { // floating: inset from every edge, children off the rounded foot corners
				if sb.X != r.X+in || sb.Y+sb.Height != r.Y+r.Height-2*in {
					t.Errorf("%s: floating sidebar %v, want inset %d, children %d off the foot", when, sb, in, 2*in)
				}
				if ib.X+ib.Width != r.X+r.Width-in || ib.Y+ib.Height != r.Y+r.Height-2*in {
					t.Errorf("%s: floating inspector %v, want inset %d, children %d off the foot", when, ib, in, 2*in)
				}
				return
			} // flush: the glass touches the top and bottom edges; the children stay below the toolbar
			top := int32(cocoa.ContentTopInset(w.Unwrap().View))
			if sb.X != r.X || sb.Y != r.Y+top || sb.Y+sb.Height != r.Y+r.Height {
				t.Errorf("%s: flush sidebar %v, want the children below the toolbar, the glass to the left edge and foot", when, sb)
			}
			if ib.X+ib.Width != r.X+r.Width || ib.Y != r.Y+top || ib.Y+ib.Height != r.Y+r.Height {
				t.Errorf("%s: flush inspector %v, want the children below the toolbar, the glass to the right edge and foot", when, ib)
			}
			glass, _, ok := cocoa.BackdropFrame(side.AsComposite().View, nil)
			if !ok || glass.Y != float64(r.Y) || glass.Y+glass.Height != float64(r.Y+r.Height) {
				t.Errorf("%s: flush sidebar glass %v, want it to the window's top and foot", when, glass)
			}
		}
		in = 7 // the macOS 26 floating glass inset
		check("floating", 7)
		look.SetSidebarFullHeight(w, true)
		look.SetInspectorFullHeight(w, true)
		look.SetBottomPanelFloating(w, false)
		app.Flush()
		in = 0
		check("flush", 0)
		look.SetSidebarFullHeight(w, false)
		look.SetInspectorFullHeight(w, false)
		look.SetBottomPanelFloating(w, true)
		app.Flush()
		in = 7
		check("floating again", 7)
	})
	if !overlayOn {
		t.Skip("panes overlay needs the macOS 26 glass look")
	}
}

// clickCounters counts the clicks the overlay window's controls got.
type clickCounters struct {
	toolbar   int              // the native toolbar item
	hitClicks int              // the button inside the bottom panel
	content   int              // mouse-downs on the gfx canvas
	hit       *gowt.ButtonView // the button inside the bottom panel
}

// runOverlayBottomApp builds the overlay test window (red sidebar/inspector/panel, green gfx
// content, panel button, toolbar item) and runs act once it showed; counters collect the clicks.
func runOverlayBottomApp(t *testing.T, counters *clickCounters, act func(*gowt.App, *gowt.Window, *gowt.Panel, *gowt.Panel, *gowt.Panel, *gowt.Panel)) bool {
	t.Helper()
	var overlayOn bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("overlay bottom")
			var side, content, insp, bottom *gowt.Panel
			var gv *viewpkg.View
			opts := []gowt.Option{gowt.Ref(&bottom)}
			if counters != nil {
				opts = append(opts, gowt.Button("Hit", gowt.Ref(&counters.hit), gowt.OnClick(func() { counters.hitClicks++ })))
			}
			opts = append(opts, gowt.Canvas(fillRed, gowt.Grow()))
			w.Add(
				gowt.VBox(gowt.Ref(&side), gowt.Canvas(fillRed, gowt.Grow())),
				gowt.VBox(gowt.Ref(&content), viewpkg.New(greenScene(), func(v *viewpkg.View) { gv = v }, gowt.Grow())),
				gowt.VBox(gowt.Ref(&insp), gowt.Canvas(fillRed, gowt.Grow())),
				gowt.VBox(opts...))
			look.SetPanesOverlay(w, true)
			look.SetSidebar(w, side, content, 220)
			look.SetInspector(w, content, insp, 260)
			look.SetBottomPanel(w, content, bottom, 160)
			if counters != nil {
				look.SetToolbar(w, look.ToolbarItem{Title: "Act", Symbol: "plus", OnClick: func() { counters.toolbar++ }})
				gv.Canvas().OnMouseDown(func(gowt.Mouse) { counters.content++ })
			}
			w.SetSize(900, 600)
			w.Show()
			app.After(time.Second, func() {
				defer w.Close()
				overlayOn, _ = w.Unwrap().GetDataKey("look.panesOverlay").(bool)
				act(app, w, side, content, insp, bottom)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return overlayOn
}

// findBottomSash returns the horizontal sash on the bottom panel's top edge: the shell child that
// is a Sash as wide as the panel and starting at its top.
func findBottomSash(t *testing.T, shell *swt.Shell, bottom *gowt.Panel) *swt.Sash {
	t.Helper()
	pb := bottom.Unwrap().GetBounds()
	for _, ch := range shell.GetChildren() {
		if _, ok := ch.Impl().(*swt.Sash); !ok {
			continue
		}
		b := ch.GetBounds()
		if b.Y == pb.Y && b.Width == pb.Width {
			return ch.Impl().(*swt.Sash)
		}
	}
	t.Fatal("no sash on the bottom panel's top edge")
	return nil
}

// postSash delivers a drag event of type to the sash at the client-area y, the way the Sash widget
// reports a drag.
func postSash(sash *swt.Sash, typ int32, y int32) {
	e := swt.NewEvent()
	e.Type = typ
	e.Y = y
	sash.NotifyListeners(typ, e)
}

// postWindowClick posts a left mouse press and release at the point (x, y) in ctl's coordinates,
// the way the user clicks; the runloop processes them in order.
func postWindowClick(ctl *swt.Composite, x, y int32) {
	win := ctl.View.Window()
	addr, _ := purego.Dlsym(purego.RTLD_DEFAULT, "objc_msgSend")
	var mouseEvent func(int64, int64, uintptr, cocoa.NSPoint, uintptr, float64, int64, uintptr, int64, int64, float64) int64
	purego.RegisterFunc(&mouseEvent, addr)
	p := ctl.View.ConvertPoint_toView_(cocoa.NSPoint{X: float64(x), Y: float64(y)}, nil)
	for _, kind := range []uintptr{1, 2} { // NSLeftMouseDown, NSLeftMouseUp
		ev := mouseEvent(cocoa.OSObjc_getClass("NSEvent"), cocoa.OSSel_registerName("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
			kind, p, 0, 0, win.WindowNumber(), 0, 1, 1, 1)
		cocoa.NSApplicationSharedApplication().PostEvent(cocoa.NewNSEventOverload1(ev), false)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// checkPaneColors asserts the strip of each pane: the content and the toolbar glass show the green
// canvas, the panes their red marker; the canvas covering a pane is what the bug looked like.
func checkPaneColors(t *testing.T, img image.Image, rects map[string][4]int32, width int32) {
	t.Helper()
	scale := float64(img.Bounds().Dx()) / float64(width)
	for name, r := range rects {
		m := meanRGB(img, r, scale)
		rr, gg, bb := m[0], m[1], m[2]
		t.Logf("%s at %v: r %.0f g %.0f b %.0f", name, r, rr, gg, bb)
		switch name {
		case "content", "toolbar":
			if gg < rr+40 || gg < bb+40 {
				t.Errorf("%s at %v is not the green canvas: r %.0f g %.0f b %.0f", name, r, rr, gg, bb)
			}
		default:
			if rr < gg+40 || rr < bb+40 {
				t.Errorf("%s at %v shows the canvas through, not the pane: r %.0f g %.0f b %.0f", name, r, rr, gg, bb)
			}
		}
	}
}
