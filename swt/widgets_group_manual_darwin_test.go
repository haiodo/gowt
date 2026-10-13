package swt

import (
	"maps"
	"os"
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func TestGroupTrimCache(t *testing.T) {
	saved := maps.Clone(groupTrims)
	t.Cleanup(func() { groupTrims = saved })
	clear(groupTrims)
	calls := 0
	measure := func(c cocoa.NSRect) cocoa.NSRect {
		calls++
		return cocoa.NSRect{X: c.X - 3, Y: c.Y - 4, Width: c.Width + 6, Height: c.Height + 20}
	}
	a := groupTrimKey{text: "a", font: "A", size: 13}
	cachedFrameFromContent(a, cocoa.NSRect{Width: 10, Height: 10}, measure)
	got := cachedFrameFromContent(a, cocoa.NSRect{X: 5, Y: 6, Width: 100, Height: 50}, measure)
	if calls != 1 || got != (cocoa.NSRect{X: 2, Y: 2, Width: 106, Height: 70}) {
		t.Fatalf("calls=%d got=%+v", calls, got)
	}
	for _, k := range []groupTrimKey{{text: "b", font: "A", size: 13}, {text: "a", font: "B", size: 13}, {text: "a", font: "A", size: 14}} {
		cachedFrameFromContent(k, cocoa.NSRect{}, measure)
	}
	if calls != 4 {
		t.Fatalf("a changed key must re-measure, calls=%d", calls)
	}
}

// Run with GOWT_GUI_TEST=1: the cached offset must equal a fresh native measurement for any content rect.
func TestGroupTrimCacheMatchesNative(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		groupCheckNative(t)
	}
	<-done
}

func groupCheckNative(t *testing.T) {
	d := NewDisplay()
	defer d.Dispose()
	shell := NewShell()
	rects := []cocoa.NSRect{{}, {Width: 10, Height: 10}, {X: 7, Y: 9, Width: 600, Height: 400}}
	long := "A very long group title that is much wider than the ten pixel content rect"
	for _, title := range []string{long, "", "Short"} {
		for _, withFont := range []bool{false, true} {
			g := NewGroup(shell, NONE)
			g.SetText(title)
			if withFont {
				g.SetFont(NewFontDeviceNameHeightStyle(d, "Helvetica", 17, BOLD))
			}
			box := castcocoaNSViewTococoaNSBox(g.View)
			clear(groupTrims)
			for _, first := range rects {
				clear(groupTrims)
				g.frameFromContentFrame(box, first)
				for _, r := range rects {
					want := g.measureFrame(box, r)
					got := g.frameFromContentFrame(box, r)
					if got != want {
						t.Errorf("title %q font %v first %+v rect %+v: cached %+v, native %+v", title, withFont, first, r, got, want)
					}
				}
			}
		}
	}
}
