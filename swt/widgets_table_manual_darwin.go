package swt

import (
	"math"

	"github.com/haiodo/gowt/internal/cocoa"
)

// syncRows tells NSTableView the row count that createItem left pending while redraw was off (tooling/port.sh).
// It is UpdateRowCount without the setRedraw pair: setRedraw_ calls it, and the pair would run setRedraw_'s tail twice.
func (this *Table) syncRows() {
	if !this.rowsPending {
		return
	}
	this.rowsPending = false
	var widget *cocoa.NSTableView = castcocoaNSViewTococoaNSTableView(this.View)
	this.ignoreSelect = true
	widget.NoteNumberOfRowsChanged()
	this.ignoreSelect = false
	widget.Tile()
}

// headerScroll is how far the rows of a Table or Tree are scrolled below their top, which lies the clip view's top
// inset above y 0.
func headerScroll(sv *cocoa.NSScrollView) float64 {
	clip := sv.ContentView()
	return clip.Bounds().Y + clipInsetTop(clip)
}

// restoreHeaderScroll scrolls the rows back to y below their top after setHeaderVisible (tooling/port.sh). The tile there
// changes the clip view's top inset by the header height but keeps its bounds origin: rows at the top went under the header.
func restoreHeaderScroll(sv *cocoa.NSScrollView, view *cocoa.NSView, y float64) {
	clip := sv.ContentView()
	view.ScrollPoint(cocoa.NSPoint{X: clip.Bounds().X, Y: y - clipInsetTop(clip)})
}

func clipInsetTop(clip *cocoa.NSClipView) float64 {
	var ins cocoa.NSRect // NSEdgeInsets has the layout of four doubles: top, left, bottom, right
	cocoa.OSObjc_msgSend_stretOverload9(&ins, clip.Id, cocoa.OSSel_registerName("contentInsets"))
	return ins.X
}

// stylePadding is the width the table style adds to the columns: 0 for SWT's plain style, 31 for the sidebar source list
// (macOS 26). getClientArea_ leaves it out (tooling/port.sh), so columns sized to the client area fit the visible width.
func stylePadding(view *cocoa.NSView) int32 {
	tv := castcocoaNSViewTococoaNSTableView(view)
	cols := tv.TableColumns()
	n := cols.Count()
	if n == 0 {
		return 0
	}
	sum := float64(n) * tv.IntercellSpacing().Width
	for i := range n {
		sum += cocoa.NewNSTableColumnOverload1(cols.ObjectAtIndex(i).Id).Width()
	}
	// The left padding is the first column's x; the right one, past the last column, measured equal to it.
	first, last := tv.RectOfColumn(0), tv.RectOfColumn(n-1)
	return int32(math.Round(last.X + last.Width + first.X - sum))
}
