package gowt

import (
	"image"
	"os"
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// responderIn reports whether the first responder of k's window is k's view or inside it (a field
// editor is). The window need not be key: over SSH the test window on the second Mac is not.
func responderIn(k *swt.Control) bool {
	win := k.View.Window()
	if win == nil {
		return false
	}
	fr := win.FirstResponder()
	if fr == nil || !fr.IsKindOfClass(cocoa.OSClass_NSView) {
		return false
	}
	return cocoa.NewNSViewOverload1(fr.Id).IsDescendantOf(k.TopView())
}

func clipWindow(app *App, n int) (*Window, *ScrollView, []*TextView) {
	var sv *ScrollView
	fields := make([]*TextView, n)
	rows := make([]Option, n)
	for i := range rows {
		rows[i] = Text(GrowX(), Ref(&fields[i]))
	}
	w := app.Window("clip")
	w.SetSize(600, 400)
	w.Add(Scroll(Grow(), Ref(&sv)))
	sv.Add(rows...)
	w.Show()
	app.Flush()
	return w, sv, fields
}

func guiClip(t *testing.T, f func(app *App)) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() { err = Run(f) })
	if err != nil {
		t.Error(err)
	}
}

// Rows far from the visible part are hidden natively and come back when scrolled to; SWT keeps them
// visible. The row with the first responder is never hidden, by a scroll or by a layout.
func TestScrollHidesFarRows(t *testing.T) {
	guiClip(t, func(app *App) {
		w, sv, fields := clipWindow(app, 200)
		defer w.Close()
		shown := func(i int) bool { return !fields[i].Unwrap().TopView().IsHidden() }
		for _, f := range fields {
			if !f.Unwrap().GetVisible() {
				t.Fatal("a row hidden off screen must stay visible to SWT")
			}
		}
		if !shown(0) || !shown(3) || shown(150) {
			t.Fatalf("at the top: rows 0, 3 shown %v %v, row 150 shown %v", shown(0), shown(3), shown(150))
		}
		if !fields[0].Focus() || !responderIn(&fields[0].Unwrap().Control) {
			t.Fatal("row 0 did not take the focus")
		}
		sv.SetOrigin(0, int(sv.Unwrap().GetMinHeight()))
		if !shown(199) || shown(50) {
			t.Errorf("at the bottom: last row shown %v, row 50 shown %v", shown(199), shown(50))
		}
		if !shown(0) || !responderIn(&fields[0].Unwrap().Control) {
			t.Errorf("scrolled away: the focused row 0 shown %v, still first responder %v", shown(0), responderIn(&fields[0].Unwrap().Control))
		}
		sv.Add(Text(GrowX())) // a layout of the content measures every row again
		if !shown(0) || !responderIn(&fields[0].Unwrap().Control) {
			t.Errorf("after Add: the focused row 0 shown %v, still first responder %v", shown(0), responderIn(&fields[0].Unwrap().Control))
		}
		sv.SetOrigin(0, 0)
		if !shown(3) || shown(199) {
			t.Errorf("back at the top: row 3 shown %v, last row shown %v", shown(3), shown(199))
		}
	})
}

// Tab, Shift+Tab and Focus reach rows that are hidden off screen, as they do without the hiding.
func TestScrollFocusHiddenRows(t *testing.T) {
	guiClip(t, func(app *App) {
		w, _, fields := clipWindow(app, 200)
		defer w.Close()
		ctl := func(i int) *swt.Control { return &fields[i].Unwrap().Control }
		first := -1
		for i := range fields {
			if ctl(i).TopView().IsHidden() {
				first = i
				break
			}
		}
		if first < 1 {
			t.Fatalf("no row is hidden at the top (first hidden %d)", first)
		}
		if !fields[first-1].Focus() || !ctl(first-1).TraverseTraversal(swt.TRAVERSE_TAB_NEXT) || !responderIn(ctl(first)) {
			t.Errorf("Tab from row %d did not reach the hidden row %d", first-1, first)
		}
		if !fields[150].Focus() || !responderIn(ctl(150)) {
			t.Fatal("Focus on the hidden row 150 failed")
		}
		if !ctl(150).TraverseTraversal(swt.TRAVERSE_TAB_PREVIOUS) || !responderIn(ctl(149)) {
			t.Error("Shift+Tab from row 150 did not reach row 149")
		}
	})
}

// Print of the content draws the rows hidden off screen too.
func TestScrollPrintHiddenRows(t *testing.T) {
	guiClip(t, func(app *App) {
		var sv *ScrollView
		w := app.Window("clip")
		w.SetSize(300, 200)
		w.Add(Scroll(Grow(), Ref(&sv)))
		for range 60 {
			sv.Add(Canvas(func(g *GC) {
				g.SetFill(RGB{R: 200})
				g.FillRect(0, 0, 300, 40)
			}, GrowX(), Size(0, 40)))
		}
		w.Show()
		app.Flush()
		defer w.Close()
		c := sv.Content().Unwrap()
		last := c.GetChildren()[59]
		if !last.TopView().IsHidden() {
			t.Fatal("the last row is not hidden")
		}
		size := c.GetSize()
		img := swt.NewImageDeviceWidthHeight(c.GetDisplay(), size.X, size.Y)
		defer img.Dispose()
		gc := swt.NewGCDrawable(img)
		c.Print(gc)
		gc.Dispose()
		data := img.GetImageData()
		y := last.GetBounds().Y + 20
		px := data.Palette.GetRGB(data.GetPixel(10, y))
		if px.Red < 150 {
			t.Errorf("printed pixel of the hidden last row: %v, want red", image.Point{10, int(y)})
		}
		if !last.TopView().IsHidden() {
			t.Error("print left the last row shown")
		}
	})
}
