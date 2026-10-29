package look_test

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// bottomWindow opens a 900 by 600 point window with a sidebar 220 points wide, the content, a bottom
// panel 160 points high and an inspector 260 points wide; check runs once it is on screen.
func bottomWindow(t *testing.T, opts []gowt.Option, check func(app *gowt.App, w *gowt.Window, side, content, panel, insp *gowt.Panel)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("bottom panel")
			var side, content, panel, insp *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&side), gowt.List([]string{"a", "b"}, gowt.Grow())),
				gowt.VBox(append([]gowt.Option{gowt.Ref(&content)}, append(opts, gowt.Label("content"))...)...),
				gowt.VBox(gowt.Ref(&panel), gowt.Padding(8), gowt.Label("panel")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(12), gowt.Label("inspector")))
			look.SetSidebar(w, side, content, 220)
			look.SetBottomPanel(w, content, panel, 160)
			look.SetInspector(w, content, insp, 260)
			w.SetSize(900, 600)
			w.Show()
			app.After(300*time.Millisecond, func() {
				defer w.Close()
				check(app, w, side, content, panel, insp)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// bottomSash is the horizontal sash of the window, on the panel's top edge.
func bottomSash(w *gowt.Window) *swt.Sash {
	for _, c := range w.Unwrap().GetChildren() {
		if s, ok := c.Impl().(*swt.Sash); ok && s.GetStyle()&swt.HORIZONTAL != 0 {
			return s
		}
	}
	return nil
}

// The panel lies under the content, as wide, between sidebar and inspector, down to the window's
// bottom edge; it hides, keeps its height while hidden, stops at 60 points and reports each change.
func TestBottomPanelLayout(t *testing.T) {
	bottomWindow(t, nil, func(app *gowt.App, w *gowt.Window, side, content, panel, insp *gowt.Panel) {
		var changes []string
		look.OnBottomPanelChange(w, func(visible bool, height int) { changes = append(changes, fmt.Sprint(visible, height)) })
		r := w.Unwrap().GetClientArea()
		sb, cb, pb, ib := side.Unwrap().GetBounds(), content.Unwrap().GetBounds(), panel.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
		if pb.X != cb.X || pb.Width != cb.Width || pb.Height != 160 || pb.Y+pb.Height != r.Y+r.Height {
			t.Errorf("panel %v under content %v in client area %v", pb, cb, r)
		}
		if gap := pb.Y - (cb.Y + cb.Height); gap < 0 || gap > 4 {
			t.Errorf("content %v ends %d points above the panel %v", cb, gap, pb)
		}
		if pb.X < sb.X+sb.Width || pb.X+pb.Width > ib.X {
			t.Errorf("panel %v runs into the sidebar %v or the inspector %v", pb, sb, ib)
		}
		if !look.BottomPanelVisible(w) || look.BottomPanelHeight(w) != 160 {
			t.Errorf("start: visible %v, height %d", look.BottomPanelVisible(w), look.BottomPanelHeight(w))
		}
		if s := bottomSash(w); s == nil || !s.GetVisible() || s.GetBounds().Width != pb.Width {
			t.Errorf("sash %v on the panel edge %v", s, pb)
		}

		look.SetBottomPanelVisible(w, false)
		app.Flush()
		cb2 := content.Unwrap().GetBounds()
		if look.BottomPanelVisible(w) || panel.Unwrap().GetVisible() || cb2.Y+cb2.Height != r.Y+r.Height {
			t.Errorf("hidden: visible %v, panel shown %v, content %v", look.BottomPanelVisible(w), panel.Unwrap().GetVisible(), cb2)
		}
		if bottomSash(w).GetVisible() {
			t.Error("a hidden panel keeps its sash")
		}
		look.SetBottomPanelVisible(w, true)
		look.SetBottomPanelHeight(w, 10)
		app.Flush()
		if got := look.BottomPanelHeight(w); got != 60 || panel.Unwrap().GetBounds().Height != 60 {
			t.Errorf("height 10: got %d, laid out %d, want 60", got, panel.Unwrap().GetBounds().Height)
		}
		if want := "[false 160 true 160 true 60]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
		if look.InspectorWidth(w) != 260 || look.SidebarWidth(w) != 220 {
			t.Errorf("the panel changed the side panes: sidebar %d, inspector %d", look.SidebarWidth(w), look.InspectorWidth(w))
		}
	})
}

// A drag delivered as the events a Sash sends: Selection with the new sash y between the mouse down
// and up.
func TestBottomPanelDragEvents(t *testing.T) {
	bottomWindow(t, nil, func(app *gowt.App, w *gowt.Window, _, content, panel, _ *gowt.Panel) {
		var changes []string
		look.OnBottomPanelChange(w, func(visible bool, height int) { changes = append(changes, fmt.Sprint(visible, height)) })
		sash := bottomSash(w)
		send := func(typ int32, y int32) int32 {
			e := swt.NewEvent()
			e.X, e.Y, e.Button = sash.GetBounds().X, y, 1
			sash.NotifyListeners(typ, e)
			return e.Y
		}
		y0 := sash.GetBounds().Y
		send(swt.MouseDown, 0)
		if got := send(swt.Selection, y0-80); got != y0-80 || look.BottomPanelHeight(w) != 240 {
			t.Errorf("drag up by 80: sash at %d, height %d", got, look.BottomPanelHeight(w))
		}
		if len(changes) != 0 {
			t.Errorf("changes during the drag: %v", changes)
		}
		r := w.Unwrap().GetClientArea()
		send(swt.Selection, r.Y+r.Height)
		if !look.BottomPanelVisible(w) || look.BottomPanelHeight(w) != 60 {
			t.Errorf("dragged to the bottom edge: visible %v, height %d; want the drag to stop at 60", look.BottomPanelVisible(w), look.BottomPanelHeight(w))
		}
		send(swt.MouseUp, 0)
		if want := "[true 60]"; fmt.Sprint(changes) != want {
			t.Errorf("changes %v, want %s", changes, want)
		}
		send(swt.MouseDown, 0)
		send(swt.Selection, r.Y)
		send(swt.MouseUp, 0)
		if ch := content.Unwrap().GetBounds().Height; !look.BottomPanelVisible(w) || ch < 100 {
			t.Errorf("dragged to the top: visible %v, content %d high", look.BottomPanelVisible(w), ch)
		}
		if pb := panel.Unwrap().GetBounds(); pb.Y+pb.Height != r.Y+r.Height {
			t.Errorf("panel %v left the bottom edge of %v", pb, r)
		}
	})
}

// A Split with Divider draws a hairline across its sash: captured with and without the option, in
// light and dark, the two pictures differ in one thin row as wide as the sash. GOWT_PIXELS_DIR keeps
// the PNGs.
func TestSplitDivider(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, mode := range []struct {
		name string
		look gowt.Appearance
	}{{"light", gowt.AppearanceLight}, {"dark", gowt.AppearanceDark}} {
		t.Run(mode.name, func(t *testing.T) {
			plain := captureSplit(t, mode.look, false, filepath.Join(dir, "divider-"+mode.name+"-off.png"))
			lined := captureSplit(t, mode.look, true, filepath.Join(dir, "divider-"+mode.name+"-on.png"))
			scale := float64(lined.img.Bounds().Dx()) / 400
			// The capture of Windows starts above the client area: the title bar row changes with the
			// focus, so only the rows of the sash can hold the hairline.
			off := int(float64(lined.img.Bounds().Dy())/scale - float64(lined.client[3]))
			from := off + int(float64(lined.sash[1])*scale) - 2
			to := off + int(float64(lined.sash[1]+lined.sash[3])*scale) + 2
			var rows []int
			for _, y := range lineRows(plain.img, lined.img, 8, int(float64(lined.sash[2])*scale*0.9)) {
				if y >= from && y <= to {
					rows = append(rows, y)
				}
			}
			t.Logf("rows changed across the sash width: %v; sash %v, client %v", rows, lined.sash, lined.client)
			if len(rows) == 0 || len(rows) > int(2*scale)+1 || rows[len(rows)-1]-rows[0] >= len(rows) {
				t.Errorf("Divider changed rows %v across the sash; want one hairline of adjacent rows", rows)
			}
		})
	}
}

// splitShot is the capture of a Split window: the picture, the sash and the client area, both in
// window points.
type splitShot struct {
	img    image.Image
	sash   [4]int32
	client [4]int32
}

// captureSplit captures a 400 by 300 point window holding a vertical Split of two empty panes, with
// or without Divider.
func captureSplit(t *testing.T, mode gowt.Appearance, divider bool, path string) splitShot {
	t.Helper()
	opts := []gowt.Option{gowt.Vertical(), gowt.Grow(), gowt.VBox(), gowt.VBox()}
	if divider {
		opts = append(opts, gowt.Divider())
	}
	var split *gowt.SplitView
	var sash, client [4]int32
	var errs []error
	var shotErr, err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("divider", gowt.Padding(0))
			w.Add(gowt.Split(append(opts, gowt.Ref(&split))...))
			w.SetSize(400, 300)
			w.Show()
			app.After(500*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				for _, ch := range split.Unwrap().GetChildren() {
					if s, ok := ch.Impl().(*swt.Sash); ok {
						b := s.GetBounds()
						sash = [4]int32{b.X, b.Y, b.Width, b.Height}
					}
				}
				r := w.Unwrap().GetClientArea()
				client = [4]int32{r.X, r.Y, r.Width, r.Height}
				shotErr = shot.WindowPNG(shotHandle(w), path)
			})
		})
	})
	for _, e := range append(errs, err, shotErr) {
		if e != nil {
			t.Fatal(e)
		}
	}
	return splitShot{decodePNG(t, path), sash, client}
}

// lineRows returns the rows where at least min pixels differ by more than levels in a channel. The
// title bar can change between two windows; a hairline is the only change across the whole sash.
func lineRows(a, b image.Image, levels, min int) []int {
	var rows []int
	r := a.Bounds().Intersect(b.Bounds())
	d := func(u, v uint32) int { return max(int(u>>8)-int(v>>8), int(v>>8)-int(u>>8)) }
	for y := r.Min.Y; y < r.Max.Y; y++ {
		n := 0
		for x := r.Min.X; x < r.Max.X; x++ {
			p0, p1, p2, _ := a.At(x, y).RGBA()
			q0, q1, q2, _ := b.At(x, y).RGBA()
			if max(d(p0, q0), d(p1, q1), d(p2, q2)) > levels {
				n++
			}
		}
		if n >= min {
			rows = append(rows, y)
		}
	}
	return rows
}
