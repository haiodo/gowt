package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func TestGroupTrimCache(t *testing.T) {
	clear(groupTrims)
	calls := 0
	measure := func(c cocoa.NSRect) cocoa.NSRect {
		calls++
		return cocoa.NSRect{X: c.X - 3, Y: c.Y - 4, Width: c.Width + 6, Height: c.Height + 20}
	}
	a := groupTrimKey{text: "a", font: 1, size: 13}
	cachedFrameFromContent(a, cocoa.NSRect{Width: 10, Height: 10}, measure)
	got := cachedFrameFromContent(a, cocoa.NSRect{X: 5, Y: 6, Width: 100, Height: 50}, measure)
	if calls != 1 || got != (cocoa.NSRect{X: 2, Y: 2, Width: 106, Height: 70}) {
		t.Fatalf("calls=%d got=%+v", calls, got)
	}
	for _, k := range []groupTrimKey{{text: "b", font: 1, size: 13}, {text: "a", font: 2, size: 13}, {text: "a", font: 1, size: 14}} {
		cachedFrameFromContent(k, cocoa.NSRect{}, measure)
	}
	if calls != 4 {
		t.Fatalf("a changed key must re-measure, calls=%d", calls)
	}
}
