package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// No Display and no window: an NSTableView set up as Table.createHandle does, with the three columns of the
// fusion-desktop sidebar (gowt-issues 4), in SWT's plain style and in the source-list style of look.SetSidebar.
func TestTableStylePadding(t *testing.T) {
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		pool := cocoa.Msg(cocoa.Msg(cocoa.OSObjc_getClass("NSAutoreleasePool"), "alloc"), "init")
		defer cocoa.Msg(pool, "drain")
		alloc := func(cls string) int64 { return cocoa.Msg(cocoa.OSObjc_getClass(cls), "alloc") }
		sv := cocoa.NewNSScrollViewOverload1(cocoa.Msg(alloc("NSScrollView"), "init"))
		defer sv.Release()
		sv.SetFrame(cocoa.NSRect{Width: 246, Height: 300})
		sv.SetHasVerticalScroller(true)
		sv.SetHasHorizontalScroller(true)
		sv.SetAutohidesScrollers(true)
		tv := cocoa.NewNSTableViewOverload1(cocoa.Msg(cocoa.Msg(alloc("NSTableView"), "init"), "autorelease"))
		tv.SetColumnAutoresizingStyle(int64(cocoa.OSNSTableViewNoColumnAutoresizing))
		tv.SetIntercellSpacing(cocoa.NSSize{Width: float64(TableCELL_GAP), Height: float64(TableCELL_GAP)})
		cocoa.Msg(tv.Id, "setStyle:", int64(cocoa.OSNSTableViewStylePlain))
		tv.SetHeaderView(nil)
		var cols []*cocoa.NSTableColumn
		for _, w := range []int32{160, 44, 30} { // TableColumn.SetWidth: the intercell gap is part of an SWT width
			c := cocoa.NewNSTableColumnOverload1(cocoa.Msg(cocoa.Msg(alloc("NSTableColumn"), "initWithIdentifier:", cocoa.NSStringString().Id), "autorelease"))
			c.SetWidth(float64(w - TableCELL_GAP))
			tv.AddTableColumn(c)
			cols = append(cols, c)
		}
		cocoa.Msg(sv.Id, "setDocumentView:", tv.Id)
		tv.Tile()
		view := &tv.NSView
		if p := stylePadding(view); p != 0 {
			t.Errorf("plain style: padding %d, want 0", p)
		}

		cocoa.Msg(tv.Id, "setStyle:", 3) // NSTableViewStyleSourceList, as cocoa.SetSourceList
		pad := stylePadding(view)
		if pad <= 0 {
			t.Fatalf("source-list style: padding %d, want > 0", pad)
		}
		// Columns wider than the clip view: the table is as wide as AppKit lays them out.
		cols[0].SetWidth(float64(1000 - TableCELL_GAP))
		tv.Tile()
		if w, want := tv.Frame().Width, float64(1000+44+30+pad); w != want {
			t.Errorf("source-list style: table %.1f wide, want %.1f (padding %d)", w, want, pad)
		}
		// The repro's Resize listener: the first column takes the client area less the other two.
		client := sv.ContentSize().Width - float64(pad)
		cols[0].SetWidth(client - 74 - float64(TableCELL_GAP))
		tv.Tile()
		if w := tv.Frame().Width; w != sv.ContentSize().Width {
			t.Errorf("table %.1f wide in a %.1f clip view: the columns do not fit", w, sv.ContentSize().Width)
		}
		for range 2 { // a resize and back: AppKit leaves the other columns alone (NoColumnAutoresizing)
			sv.SetFrame(cocoa.NSRect{Width: 300, Height: 300})
			tv.Tile()
			sv.SetFrame(cocoa.NSRect{Width: 246, Height: 300})
			tv.Tile()
		}
		if w1, w2 := cols[1].Width(), cols[2].Width(); w1 != 43 || w2 != 29 {
			t.Errorf("columns 2 and 3 are %.1f and %.1f, want 43 and 29", w1, w2)
		}
	}
	<-done
}
