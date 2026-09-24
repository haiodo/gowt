// Hand-written: exercises Round 5's Layout/FillLayout/RowLayout/FillData/RowData translation -
// pure value/cascade-wiring checks only, no live Cocoa view needed (ComputeSize itself calls
// into a real Control's NSView, out of scope for a unit test at this round).
package swt

import "testing"

func TestRowDataDefaults(t *testing.T) {
	d := NewRowData()
	if d.Width != SWTDEFAULT || d.Height != SWTDEFAULT || d.Exclude {
		t.Fatalf("got %+v", d)
	}
	d2 := NewRowDataWidthHeight(50, 40)
	if d2.Width != 50 || d2.Height != 40 {
		t.Fatalf("got %+v", d2)
	}
	if got := d2.String(); got != "RowData {width=50 height=40}" {
		t.Fatalf("got %q", got)
	}
}

func TestFillDataFlushCache(t *testing.T) {
	d := newFillData()
	if d.defaultWidth != -1 || d.defaultHeight != -1 || d.currentWidth != -1 || d.currentHeight != -1 {
		t.Fatalf("got %+v", d)
	}
	d.defaultWidth, d.defaultHeight, d.currentWidth, d.currentHeight = 10, 20, 30, 40
	d.FlushCache()
	if d.defaultWidth != -1 || d.defaultHeight != -1 || d.currentWidth != -1 || d.currentHeight != -1 {
		t.Fatalf("FlushCache did not reset: %+v", d)
	}
}

// FillLayout/RowLayout's own impl-cascade wiring: this.impl must resolve back to the concrete
// subclass, and each carries Layout's promoted FlushCache without needing a live Composite.
func TestLayoutCascadeWiring(t *testing.T) {
	fl := NewFillLayout()
	if _, ok := fl.impl.(*FillLayout); !ok {
		t.Fatalf("FillLayout.impl = %T, want *FillLayout", fl.impl)
	}
	if fl.Type != SWTHORIZONTAL || fl.MarginWidth != 0 || fl.Spacing != 0 {
		t.Fatalf("got %+v", fl)
	}
	// FillLayout overrides FlushCache for real (needs a live Control) - only the base Layout's
	// own default (used by a layout that doesn't override it) is pure enough to test here.
	base := &Layout{}
	base.impl = base
	if base.FlushCache(nil) {
		t.Fatal("Layout.FlushCache should default to false")
	}

	rl := NewRowLayout()
	if _, ok := rl.impl.(*RowLayout); !ok {
		t.Fatalf("RowLayout.impl = %T, want *RowLayout", rl.impl)
	}
	if rl.Type != SWTHORIZONTAL || rl.Wrap != true || rl.Pack != true {
		t.Fatalf("got %+v", rl)
	}
}
