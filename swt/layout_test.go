// Hand-written: exercises Round 5's Layout/FillLayout/RowLayout/FillData/RowData translation -
// pure value/cascade-wiring checks only, no live Cocoa view needed (ComputeSize itself calls
// into a real Control's NSView, out of scope for a unit test at this round).
package swt

import "testing"

func TestRowDataDefaults(t *testing.T) {
	d := NewRowData()
	if d.Width != DEFAULT || d.Height != DEFAULT || d.Exclude {
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
	if fl.Type != HORIZONTAL || fl.MarginWidth != 0 || fl.Spacing != 0 {
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
	if rl.Type != HORIZONTAL || rl.Wrap != true || rl.Pack != true {
		t.Fatalf("got %+v", rl)
	}
}

// Ported from SWT's Test_org_eclipse_swt_layout_GridData - pure field/constant checks, no
// live Cocoa view needed.
func TestGridDataConstructor(t *testing.T) {
	data := NewGridData()
	if data.VerticalAlignment != GridDataCENTER || data.HorizontalAlignment != GridDataBEGINNING {
		t.Fatalf("got %+v", data)
	}
	if data.WidthHint != DEFAULT || data.HeightHint != DEFAULT {
		t.Fatalf("got %+v", data)
	}
	if data.HorizontalIndent != 0 || data.HorizontalSpan != 1 || data.VerticalSpan != 1 {
		t.Fatalf("got %+v", data)
	}
	if data.GrabExcessHorizontalSpace || data.GrabExcessVerticalSpace {
		t.Fatalf("got %+v", data)
	}
}

func TestGridDataConstructorStyle(t *testing.T) {
	data := NewGridDataStyle(GridDataFILL_BOTH)
	if data.VerticalAlignment != GridDataFILL || data.HorizontalAlignment != GridDataFILL {
		t.Fatalf("got %+v", data)
	}
	if !data.GrabExcessHorizontalSpace || !data.GrabExcessVerticalSpace {
		t.Fatalf("got %+v", data)
	}
}

func TestGridDataConstructorWidthHeight(t *testing.T) {
	data := NewGridDataWidthHeight(100, 100)
	if data.WidthHint != 100 || data.HeightHint != 100 {
		t.Fatalf("got %+v", data)
	}
}

// Ported from SWT's Test_org_eclipse_swt_layout_FormAttachment - the ctors only store the
// Control pointer, so a bare zero-value *Shell stands in for the original's live one.
func TestFormAttachmentConstructors(t *testing.T) {
	shell := &Shell{}
	if a := NewFormAttachmentNumerator(50); a == nil {
		t.Fatal("NewFormAttachmentNumerator returned nil")
	}
	if a := NewFormAttachmentNumeratorOffset(50, 10); a == nil {
		t.Fatal("NewFormAttachmentNumeratorOffset returned nil")
	}
	if a := NewFormAttachmentNumeratorDenominatorOffset(50, 100, 10); a == nil {
		t.Fatal("NewFormAttachmentNumeratorDenominatorOffset returned nil")
	}
	if a := NewFormAttachmentControl(&shell.Control); a == nil {
		t.Fatal("NewFormAttachmentControl returned nil")
	}
	if a := NewFormAttachmentControlOffset(&shell.Control, 10); a == nil {
		t.Fatal("NewFormAttachmentControlOffset returned nil")
	}
	if a := NewFormAttachmentControlOffsetAlignment(&shell.Control, 10, LEFT); a == nil {
		t.Fatal("NewFormAttachmentControlOffsetAlignment returned nil")
	}
}

func TestFormAttachmentString(t *testing.T) {
	a := NewFormAttachmentNumerator(50)
	if got := a.String(); got == "" {
		t.Fatal("FormAttachment.String() returned empty")
	}
}
