package swt

import (
	"math"
	"os"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt/internal/cocoa"
)

// Table and Tree are view-based NSTableView/NSOutlineView by default (docs/view-based-table.md); GOWT_VIEW_TABLE=0 selects the
// cell-based path (cell-based drawing adds ~227 MB of GPU memory while cells update). Opt-out is kept for one release.
var viewTableEnabled = os.Getenv("GOWT_VIEW_TABLE") != "0"

// Cell views draw through the table's shared data cell (the same EraseItem/MeasureItem/PaintItem flow as drawInteriorWithFrame).
var (
	vtCellClass  int64
	vtCheckClass int64
	vtDrawing    = map[int64]bool{} // cell views inside drawRect: invalidating them from there would loop
	vtViewCalls  int                // viewForTableColumn calls that reached vtMakeView; read by the GUI test
)

func init() {
	if viewTableEnabled {
		cocoa.OnClassRegistered(vtClassRegistered)
	}
}

func vtClassRegistered(cls int64) {
	switch cocoa.OSClass_getName(cls) {
	case "SWTTableView":
		vtInstall(cls, false)
	case "SWTOutlineView":
		vtInstall(cls, true)
	}
}

func vtInstall(cls int64, outline bool) {
	if outline {
		cocoa.OSClass_addMethod(cls, cocoa.OSSel_registerName("outlineView:viewForTableColumn:item:"), cocoa.NewCallbackN(5, vtOutlineViewFor), "@@:@@@")
	} else {
		cocoa.OSClass_addMethod(cls, cocoa.OSSel_registerName("tableView:viewForTableColumn:row:"), cocoa.NewCallbackN(5, vtTableViewFor), "@@:@@q")
	}
	vtWrapNeedsDisplayInRect(cls)
	vtWrapKeyDown(cls)
}

// vtReplace installs imp for sel on cls only (class_addMethod fails when cls already has its own method; an inherited one
// must not be rewritten in place) and returns the implementation it replaced.
func vtReplace(cls int64, sel int64, imp int64, types string) uintptr {
	orig := cocoa.OSClass_getMethodImplementation(cls, sel)
	if !cocoa.OSClass_addMethod(cls, sel, imp, types) {
		cocoa.OSMethod_setImplementation(cocoa.OSClass_getInstanceMethod(cls, sel), imp)
	}
	return uintptr(orig)
}

// A change of an item redraws the table's rect; cell views are separate layers and do not follow that invalidation.
func vtWrapNeedsDisplayInRect(cls int64) {
	var orig func(self, sel uintptr, r cocoa.NSRect)
	imp := cocoa.OSCALLBACK_setNeedsDisplayInRect_(cocoa.NewCallbackN(3, func(a []int64) int64 {
		var r cocoa.NSRect
		cocoa.OSMemmoveOverload7(&r, a[2], int64(cocoa.NSRectSizeof))
		orig(uintptr(a[0]), uintptr(a[1]), r)
		vtInvalidateCells(a[0], r)
		return 0
	}))
	purego.RegisterFunc(&orig, vtReplace(cls, cocoa.OSSel_setNeedsDisplayInRect_, imp, "v@:{CGRect={CGPoint=dd}{CGSize=dd}}"))
}

func vtInvalidateCells(table int64, r cocoa.NSRect) {
	rows := cocoa.NewNSViewOverload1(table).Subviews()
	for i := int64(0); i < rows.Count(); i++ {
		row := cocoa.NewNSViewOverload1(rows.ObjectAtIndex(i).Id)
		if !vtIntersects(row.Frame(), r) {
			continue
		}
		cells := row.Subviews()
		for j := int64(0); j < cells.Count(); j++ {
			if cell := cells.ObjectAtIndex(j).Id; !vtDrawing[cell] {
				cocoa.Msg(cell, "setNeedsDisplay:", 1)
			}
		}
	}
}

func vtIntersects(a, b cocoa.NSRect) bool {
	return a.X < b.X+b.Width && b.X < a.X+a.Width && a.Y < b.Y+b.Height && b.Y < a.Y+a.Height
}

// NSTableView toggles a check box on Space only through its cell; with views the key is handled here.
func vtWrapKeyDown(cls int64) {
	var orig func(self, sel, event uintptr)
	imp := cocoa.NewCallbackN(3, func(a []int64) int64 {
		ev := cocoa.NewNSEventOverload1(a[2])
		w := DisplayGetWidget(a[0])
		var seen *vtSpaceListener
		if c := vtControl(w); c != nil && ev.Characters().GetString() == " " {
			seen = &vtSpaceListener{}
			c.AddListener(KeyDown, seen)
			defer func() {
				if !c.IsDisposed() {
					c.RemoveListener(KeyDown, seen)
				}
			}()
		}
		orig(uintptr(a[0]), uintptr(a[1]), uintptr(a[2]))
		if seen != nil && seen.plain {
			vtToggleSelected(a[0])
		}
		return 0
	})
	purego.RegisterFunc(&orig, vtReplace(cls, cocoa.OSSel_keyDown_, imp, "v@:@"))
}

// vtSpaceListener is added after the application's listeners: it sees a Space KeyDown that none of them cancelled and that
// carried no Ctrl/Alt/Command, which is when the cell path reached NSTableView's own Space handling.
type vtSpaceListener struct{ plain bool }

func (l *vtSpaceListener) HandleEvent(e *Event) {
	l.plain = e.Doit && e.Character == ' ' && (e.StateMask&(CTRL|ALT|COMMAND)) == 0
}

func vtControl(w *Widget) *Control {
	if t := vtAsTable(w); t != nil {
		return &t.Control
	}
	if t := vtAsTree(w); t != nil {
		return &t.Control
	}
	return nil
}

func vtToggleSelected(view int64) {
	w := DisplayGetWidget(view)
	if t := vtAsTable(w); t != nil {
		t.vtToggleSelected()
	} else if t := vtAsTree(w); t != nil {
		t.vtToggleSelected()
	}
}

func vtAsTable(w *Widget) *Table {
	if w == nil {
		return nil
	}
	if t, ok := w.Impl().(interface{ AsTable() *Table }); ok {
		return t.AsTable()
	}
	return nil
}

func vtAsTree(w *Widget) *Tree {
	if w == nil {
		return nil
	}
	if t, ok := w.Impl().(interface{ AsTree() *Tree }); ok {
		return t.AsTree()
	}
	return nil
}

func vtAsList(w *Widget) *List {
	if w == nil {
		return nil
	}
	if l, ok := w.Impl().(interface{ AsList() *List }); ok {
		return l.AsList()
	}
	return nil
}

func vtTableViewFor(a []int64) int64 {
	w := DisplayGetWidget(a[0])
	if l := vtAsList(w); l != nil {
		// List shares the SWTTableView class with Table; a nil view leaves the cell blank.
		if a[3] == 0 || a[4] < 0 || a[4] >= int64(l.itemCount) {
			return 0
		}
		return vtMakeView(a[2], false)
	}
	t := vtAsTable(w)
	if t == nil || a[3] == 0 || a[4] < 0 || a[4] >= int64(t.itemCount) {
		return 0
	}
	return vtMakeView(a[2], t.checkColumn != nil && a[3] == t.checkColumn.Id)
}

func vtOutlineViewFor(a []int64) int64 {
	t := vtAsTree(DisplayGetWidget(a[0]))
	if t == nil || a[3] == 0 || a[4] == 0 {
		return 0
	}
	return vtMakeView(a[2], t.checkColumn != nil && a[3] == t.checkColumn.Id)
}

func vtMakeView(tableView int64, check bool) int64 {
	vtViewCalls++
	cell, checkCls := vtClasses()
	name, cls := "SWTCell", cell
	if check {
		name, cls = "SWTCheck", checkCls
	}
	ident := cocoa.NSStringID(name)
	v := cocoa.Msg(tableView, "makeViewWithIdentifier:owner:", ident, tableView)
	if v == 0 {
		v = cocoa.Msg(cocoa.Msg(cocoa.Msg(cls, "alloc"), "init"), "autorelease")
		cocoa.Msg(v, "setIdentifier:", ident)
		// NSViewLayerContentsRedrawDuringViewResize: a column resize must redraw the text, not stretch the layer.
		cocoa.Msg(v, "setLayerContentsRedrawPolicy:", 2)
		if !check {
			// One huge rect instead of tracking the cell size: the view is asked for the text only when the pointer rests on it.
			cocoa.NewNSViewOverload1(v).AddToolTipRect(cocoa.NSRect{Width: 1e5, Height: 1e5}, cocoa.NewidOverload1(v), 0)
		}
	}
	return v
}

func vtClasses() (cell int64, check int64) {
	if vtCellClass != 0 {
		return vtCellClass, vtCheckClass
	}
	super := cocoa.OSObjc_getClass("NSTableCellView")
	cls := cocoa.OSObjc_allocateClassPair(super, "SWTTableCellView", 0)
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_drawRect_, cocoa.OSCALLBACK_drawRect_(cocoa.NewCallbackN(3, vtDrawRect)), "v@:{CGRect={CGPoint=dd}{CGSize=dd}}")
	// SWT paints in the flipped coordinates of NSTableView.
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_isFlipped, cocoa.NewCallbackN(2, func(a []int64) int64 { return 1 }), "B@:")
	var superStyle func(self, sel, style uintptr)
	purego.RegisterFunc(&superStyle, uintptr(cocoa.OSClass_getMethodImplementation(super, cocoa.OSSel_setBackgroundStyle_)))
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_setBackgroundStyle_, cocoa.NewCallbackN(3, func(a []int64) int64 {
		superStyle(uintptr(a[0]), uintptr(a[1]), uintptr(a[2]))
		cocoa.Msg(a[0], "setNeedsDisplay:", 1)
		return 0
	}), "v@:q")
	vtAddAccessibility(cls)
	// NSPoint arrives in FP registers on arm64: vtToolTip reads only self.
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_view_stringForToolTip_point_userData_, cocoa.NewCallbackN(6, vtToolTip), "@@:@q{NSPoint}^v")
	cocoa.OSObjc_registerClassPair(cls)
	chk := cocoa.OSObjc_allocateClassPair(cls, "SWTTableCheckView", 0)
	cocoa.OSClass_addMethod(chk, cocoa.OSSel_mouseDown_, cocoa.NewCallbackN(3, vtCheckMouseDown), "v@:@")
	// The cell path gets MouseUp from the tracking loop of NSTableView; the check view takes no tracking, so AppKit sends mouseUp: to it.
	cocoa.OSClass_addMethod(chk, cocoa.OSSel_mouseUp_, cocoa.NewCallbackN(3, vtCheckMouseUp), "v@:@")
	vtAddCheckAccessibility(chk)
	cocoa.OSObjc_registerClassPair(chk)
	vtCellClass, vtCheckClass = cls, chk
	return cls, chk
}

func vtDrawRect(a []int64) int64 {
	view := a[0]
	w := DisplayLookupWidget(view, 0)
	check := cocoa.OSObject_getClass(view) == vtCheckClass
	vtDrawing[view] = true
	defer delete(vtDrawing, view)
	if t := vtAsTable(w); t != nil {
		t.vtDrawCell(view, check)
	} else if t := vtAsTree(w); t != nil {
		t.vtDrawCell(view, check)
	} else if l := vtAsList(w); l != nil {
		l.vtDrawCell(view)
	}
	return 0
}

func vtCheckMouseDown(a []int64) int64 {
	w := DisplayLookupWidget(a[0], 0)
	if t := vtAsTable(w); t != nil {
		t.vtCheckMouseDown(a[0], a[2])
	} else if t := vtAsTree(w); t != nil {
		t.vtCheckMouseDown(a[0], a[2])
	}
	return 0
}

// vtOwner is the Table or Tree that owns a cell view.
type vtOwner interface {
	vtValue(view int64) (int64, bool)
	vtToggle(view int64) bool
	vtToolTip(view int64) string
}

func vtOwnerOf(view int64) vtOwner {
	w := DisplayLookupWidget(view, 0)
	if t := vtAsTable(w); t != nil {
		return t
	}
	if t := vtAsTree(w); t != nil {
		return t
	}
	return nil
}

// Unconditional like the cell path: there Display sends MouseUp to the tracking control whatever the MouseDown doit was.
func vtCheckMouseUp(a []int64) int64 {
	if c := vtControl(DisplayLookupWidget(a[0], 0)); c != nil && !c.IsDisposed() {
		c.impl.mouseEvent_(c.View.Id, cocoa.OSSel_mouseUp_, a[2], MouseUp)
	}
	return 0
}

func vtMethodAdder(cls int64) func(name string, nargs int, fn func(a []int64) int64, types string) {
	return func(name string, nargs int, fn func(a []int64) int64, types string) {
		cocoa.OSClass_addMethod(cls, cocoa.OSSel_registerName(name), cocoa.NewCallbackN(nargs, fn), types)
	}
}

// Accessibility of the cell views: NSTableCellView without a text field exposes nothing, the text is drawn by the shared data cell.
func vtAddAccessibility(cls int64) {
	add := vtMethodAdder(cls)
	add("isAccessibilityElement", 2, func(a []int64) int64 { return 1 }, "B@:")
	add("accessibilityRole", 2, func(a []int64) int64 { return cocoa.NSStringID("AXCell") }, "@@:")
	add("accessibilityLabel", 2, func(a []int64) int64 {
		if cocoa.OSObject_getClass(a[0]) == vtCheckClass {
			return 0
		}
		if o := vtOwnerOf(a[0]); o != nil {
			if v, ok := o.vtValue(a[0]); ok {
				return v
			}
		}
		return 0
	}, "@@:")
}

func vtAddCheckAccessibility(cls int64) {
	add := vtMethodAdder(cls)
	add("accessibilityRole", 2, func(a []int64) int64 { return cocoa.NSStringID("AXCheckBox") }, "@@:")
	// The value of the check column is the NSNumber control state (off 0, on 1, mixed -1), the same one VoiceOver reads from a button.
	add("accessibilityValue", 2, func(a []int64) int64 {
		if o := vtOwnerOf(a[0]); o != nil {
			if v, ok := o.vtValue(a[0]); ok {
				return v
			}
		}
		return 0
	}, "@@:")
	add("accessibilityPerformPress", 2, func(a []int64) int64 {
		if o := vtOwnerOf(a[0]); o != nil {
			if o.vtToggle(a[0]) {
				return 1
			}
		}
		return 0
	}, "B@:")
}

// view:stringForToolTip:point:userData: of a cell view: the text as a tooltip when the cell truncates it (the cell path's expansion tooltip).
func vtToolTip(a []int64) int64 {
	if o := vtOwnerOf(a[0]); o != nil {
		if s := o.vtToolTip(a[0]); s != "" {
			return cocoa.NSStringID(s)
		}
	}
	return 0
}

// vtDrawContext draws in the table's coordinates, frame (the cell in the table) mapped onto the view.
// Not the view's current place: while an expand animates, the rows below still sit at their old y.
func vtDrawContext(view *cocoa.NSView, frame cocoa.NSRect, draw func()) {
	b := view.Bounds()
	ctx := cocoa.NSGraphicsContextCurrentContext()
	ctx.SaveGraphicsState()
	transform := cocoa.NSAffineTransformTransform()
	transform.TranslateXBy(b.X-frame.X, b.Y-frame.Y)
	transform.Concat()
	draw()
	ctx.RestoreGraphicsState()
}

func (this *Table) vtDrawCell(view int64, check bool) {
	if this.IsDisposed() {
		return
	}
	tv := castcocoaNSViewTococoaNSTableView(this.View)
	row := cocoa.Msg(tv.Id, "rowForView:", view)
	col := cocoa.Msg(tv.Id, "columnForView:", view)
	if row < 0 || col < 0 || row >= int64(this.itemCount) {
		return
	}
	column := tv.TableColumns().ObjectAtIndex(col).Id
	v := cocoa.NewNSViewOverload1(view)
	if check {
		if this.buttonCell == nil {
			return
		}
		this.buttonCell.SetObjectValue(cocoa.NewidOverload1(this.tableView_objectValueForTableColumn_row_(tv.Id, 0, tv.Id, column, row)))
		this.buttonCell.DrawInteriorWithFrame(v.Bounds(), v)
		return
	}
	cell := this.dataCell
	value := this.tableView_objectValueForTableColumn_row_(tv.Id, 0, tv.Id, column, row)
	if this.IsDisposed() || cell == nil {
		return
	}
	cell.SetObjectValue(cocoa.NewidOverload1(value))
	selected := tv.IsRowSelected(row)
	cell.SetHighlighted(selected)
	this.tableView_willDisplayCell_forTableColumn_row_(tv.Id, 0, tv.Id, cell.Id, column, row)
	// A source list draws the selected row's text and icon in the accent colour, as List.vtDrawCell.
	// A font with synthetic traits keeps the colour of its attributed string.
	if selected && cocoa.Msg(tv.Id, "style") == vtStyleSourceList {
		accent := cocoa.Msg(cocoa.OSObjc_getClass("NSColor"), "controlAccentColor")
		cocoa.Msg(cell.Id, "setTextColor:", accent)
		if img := cell.Image(); img != nil {
			cell.SetImage(vtTinted(img, col, accent))
		}
	}
	frame := tv.FrameOfCellAtColumn(col, row)
	// SWT clips cell text; Finder and AppKit lists end it with an ellipsis.
	cell.SetLineBreakMode(int64(cocoa.OSNSLineBreakByTruncatingTail))
	vtDrawContext(v, frame, func() { cell.DrawInteriorWithFrame(frame, this.View) })
}

func (this *List) vtDrawCell(view int64) {
	if this.IsDisposed() {
		return
	}
	tv := castcocoaNSViewTococoaNSTableView(this.View)
	row := cocoa.Msg(tv.Id, "rowForView:", view)
	col := cocoa.Msg(tv.Id, "columnForView:", view)
	if row < 0 || col < 0 || row >= int64(this.itemCount) {
		return
	}
	cell := this.column.DataCell()
	cell.SetObjectValue(cocoa.NewidOverload1(this.tableView_objectValueForTableColumn_row_(tv.Id, 0, tv.Id, this.column.Id, row)))
	selected := tv.IsRowSelected(row)
	cell.SetHighlighted(selected)
	cellFrame := tv.FrameOfCellAtColumn(col, row)
	frame := cellFrame
	// The text cell draws from the top of its frame; a row taller than the text (source list) centres it.
	if h := cell.CellSize().Height; h < frame.Height {
		frame.Y += math.Floor((frame.Height - h) / 2)
		frame.Height = h
	}
	// A source list (look.SetSidebar) selects with a grey pill and accent-coloured text, as Finder does.
	if selected && cocoa.Msg(tv.Id, "style") == vtStyleSourceList {
		old := cocoa.Msg(cell.Id, "textColor")
		cocoa.Msg(old, "retain")
		defer func() {
			cocoa.Msg(cell.Id, "setTextColor:", old)
			cocoa.Msg(old, "release")
		}()
		cell.SetHighlighted(false)
		cocoa.Msg(cell.Id, "setTextColor:", cocoa.Msg(cocoa.OSObjc_getClass("NSColor"), "controlAccentColor"))
	}
	cell.SetLineBreakMode(int64(cocoa.OSNSLineBreakByTruncatingTail))
	vtDrawContext(cocoa.NewNSViewOverload1(view), cellFrame, func() { cell.DrawInteriorWithFrame(frame, this.View) })
}

const vtStyleSourceList = 3 // NSTableViewStyleSourceList

// vtTinted returns img in color, autoreleased: one bitmap per representation of img keeps it sharp
// at every scale. Only the glyph of the leading column (col 0) is tinted, as Finder tints the row symbol
// only; a badge in a later column and any image that is not a glyph (vtGlyph) come back as they are.
func vtTinted(img *cocoa.NSImage, col int64, color int64) *cocoa.NSImage {
	if col != 0 {
		return img
	}
	template := cocoa.Msg(img.Id, "isTemplate")&0xff != 0
	size := img.Size()
	rect := cocoa.NSRect{Width: size.Width, Height: size.Height}
	out := castcocoaNSObjectTococoaNSImage(cocoa.NewNSImage().Alloc()).InitWithSize(size)
	reps := img.Representations()
	for i := int64(0); i < reps.Count(); i++ {
		src := cocoa.NewNSImageRepOverload1(reps.ObjectAtIndex(i).Id)
		w, h := src.PixelsWide(), src.PixelsHigh()
		if w <= 0 || h <= 0 { // a vector representation: one pixel per point
			w, h = int64(math.Ceil(size.Width)), int64(math.Ceil(size.Height))
		}
		rep := castcocoaNSObjectTococoaNSBitmapImageRep(cocoa.NewNSBitmapImageRep().Alloc())
		rep = rep.InitWithBitmapDataPlanes(0, w, h, 8, 4, true, false, cocoa.OSNSDeviceRGBColorSpace_, 0, w*4, 32)
		rep.SetSize(size)
		cocoa.NSGraphicsContextStatic_saveGraphicsState()
		ctx := cocoa.NSGraphicsContextGraphicsContextWithBitmapImageRep(rep)
		cocoa.NSGraphicsContextSetCurrentContext(ctx)
		src.DrawInRect(rect)
		ctx.FlushGraphics()
		if !template && !vtGlyph(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), rep.BitmapData())), w*h*4)) {
			cocoa.NSGraphicsContextStatic_restoreGraphicsState()
			rep.Release()
			out.Release()
			return img
		}
		ctx.SetCompositingOperation(int64(cocoa.OSNSCompositingOperationSourceAtop))
		cocoa.NewNSColorOverload1(color).SetFill()
		cocoa.NSBezierPathFillRect(rect)
		cocoa.NSGraphicsContextStatic_restoreGraphicsState()
		out.AddRepresentation(upcastcocoaNSBitmapImageRepTococoaNSImageRep(rep))
		rep.Release()
	}
	cocoa.Msg(out.Id, "autorelease")
	return out
}

// vtGlyph reports premultiplied RGBA pixels that are one grey with alpha, as a template image is: icons.Get
// makes bitmaps, not template NSImages. A coloured, multi-tone or opaque image is not a glyph.
func vtGlyph(pix []byte) bool {
	const tolerance = 24
	lo, hi := 255, 0
	seeThrough := false
	for i := 0; i+3 < len(pix); i += 4 {
		a := int(pix[i+3])
		if a == 0 {
			seeThrough = true
		}
		if a < 128 { // edge pixels lose precision when unpremultiplied
			continue
		}
		r, g, b := int(pix[i])*255/a, int(pix[i+1])*255/a, int(pix[i+2])*255/a
		if max(r, g, b)-min(r, g, b) > tolerance {
			return false
		}
		lo, hi = min(lo, r, g, b), max(hi, r, g, b)
	}
	return seeThrough && lo <= hi && hi-lo <= tolerance
}

func (this *Table) vtCheckMouseDown(view int64, event int64) {
	tv := castcocoaNSViewTococoaNSTableView(this.View)
	row := cocoa.Msg(tv.Id, "rowForView:", view)
	if row < 0 || row >= int64(this.itemCount) {
		return
	}
	// The mouseDown: of the table view without the NSTableView part: a click on a check box does not select the row.
	if !this.impl.mouseEvent_(tv.Id, cocoa.OSSel_mouseDown_, event, MouseDown) || this.IsDisposed() || row >= int64(this.itemCount) {
		return
	}
	this.ToggleCheckedItem(this._getItem(int32(row)), row)
}

func (this *Table) vtLocate(view int64) (tv *cocoa.NSTableView, row, col, column int64, ok bool) {
	if this.IsDisposed() {
		return
	}
	tv = castcocoaNSViewTococoaNSTableView(this.View)
	row = cocoa.Msg(tv.Id, "rowForView:", view)
	col = cocoa.Msg(tv.Id, "columnForView:", view)
	if row < 0 || col < 0 || row >= int64(this.itemCount) {
		return
	}
	return tv, row, col, tv.TableColumns().ObjectAtIndex(col).Id, true
}

func (this *Table) vtValue(view int64) (int64, bool) {
	tv, row, _, column, ok := this.vtLocate(view)
	if !ok {
		return 0, false
	}
	return this.tableView_objectValueForTableColumn_row_(tv.Id, 0, tv.Id, column, row), true
}

func (this *Table) vtToggle(view int64) bool {
	_, row, _, _, ok := this.vtLocate(view)
	if !ok || !this.IsEnabled() {
		return false
	}
	item := this._getItem(int32(row))
	if item == nil || item.IsDisposed() {
		return false
	}
	this.ToggleCheckedItem(item, row)
	return true
}

func (this *Table) vtToolTip(view int64) string {
	tv, row, _, column, ok := this.vtLocate(view)
	cell := this.dataCell
	if !ok || this.toolTipText != "" || cell == nil {
		return ""
	}
	value := this.tableView_objectValueForTableColumn_row_(tv.Id, 0, tv.Id, column, row)
	cell.SetObjectValue(cocoa.NewidOverload1(value))
	this.tableView_willDisplayCell_forTableColumn_row_(tv.Id, 0, tv.Id, cell.Id, column, row)
	if cell.CellSize().Width <= cocoa.NewNSViewOverload1(view).Bounds().Width {
		return ""
	}
	return cocoa.NewNSStringOverload1(value).GetString()
}

func (this *Table) vtToggleSelected() {
	if this.IsDisposed() || (this.style&CHECK) == 0 {
		return
	}
	selection := castcocoaNSViewTococoaNSTableView(this.View).SelectedRowIndexes()
	count := int32(selection.Count())
	indices := make([]int64, count)
	selection.GetIndexes(indices, int64(count), int64(0))
	for _, i := range indices {
		this.ToggleCheckedItem(this._getItem(int32(i)), i)
	}
}

func (this *Tree) vtDrawCell(view int64, check bool) {
	if this.IsDisposed() {
		return
	}
	ov := castcocoaNSViewTococoaNSOutlineView(this.View)
	row := cocoa.Msg(ov.Id, "rowForView:", view)
	col := cocoa.Msg(ov.Id, "columnForView:", view)
	if row < 0 || col < 0 {
		return
	}
	column := ov.TableColumns().ObjectAtIndex(col).Id
	itemID := ov.ItemAtRow(row).Id
	v := cocoa.NewNSViewOverload1(view)
	if check {
		if this.buttonCell == nil {
			return
		}
		this.buttonCell.SetObjectValue(cocoa.NewidOverload1(this.outlineView_objectValueForTableColumn_byItem_(ov.Id, 0, ov.Id, column, itemID)))
		this.buttonCell.DrawInteriorWithFrame(v.Bounds(), v)
		return
	}
	cell := this.dataCell
	value := this.outlineView_objectValueForTableColumn_byItem_(ov.Id, 0, ov.Id, column, itemID)
	if this.IsDisposed() || cell == nil {
		return
	}
	cell.SetObjectValue(cocoa.NewidOverload1(value))
	cell.SetHighlighted(ov.IsRowSelected(row))
	this.outlineView_willDisplayCell_forTableColumn_item_(ov.Id, 0, ov.Id, cell.Id, column, itemID)
	frame := ov.FrameOfCellAtColumn(col, row)
	cell.SetLineBreakMode(int64(cocoa.OSNSLineBreakByTruncatingTail))
	vtDrawContext(v, frame, func() { cell.DrawInteriorWithFrame(frame, this.View) })
}

func (this *Tree) vtCheckMouseDown(view int64, event int64) {
	ov := castcocoaNSViewTococoaNSOutlineView(this.View)
	row := cocoa.Msg(ov.Id, "rowForView:", view)
	if row < 0 {
		return
	}
	if !this.impl.mouseEvent_(ov.Id, cocoa.OSSel_mouseDown_, event, MouseDown) || this.IsDisposed() {
		return
	}
	// A MouseDown listener may have disposed or moved the item: resolve it again.
	row = cocoa.Msg(ov.Id, "rowForView:", view)
	if row < 0 || row >= ov.NumberOfRows() {
		return
	}
	itemID := ov.ItemAtRow(row).Id
	if item := this.display.GetWidget(itemID); item == nil || item.IsDisposed() {
		return
	}
	this.outlineView_setObjectValue_forTableColumn_byItem_(ov.Id, 0, ov.Id, 0, this.checkColumn.Id, itemID)
}

func (this *Tree) vtLocate(view int64) (ov *cocoa.NSOutlineView, col, column, item int64, ok bool) {
	if this.IsDisposed() {
		return
	}
	ov = castcocoaNSViewTococoaNSOutlineView(this.View)
	row := cocoa.Msg(ov.Id, "rowForView:", view)
	col = cocoa.Msg(ov.Id, "columnForView:", view)
	if row < 0 || col < 0 || row >= ov.NumberOfRows() {
		return
	}
	return ov, col, ov.TableColumns().ObjectAtIndex(col).Id, ov.ItemAtRow(row).Id, true
}

func (this *Tree) vtValue(view int64) (int64, bool) {
	ov, _, column, item, ok := this.vtLocate(view)
	if !ok {
		return 0, false
	}
	return this.outlineView_objectValueForTableColumn_byItem_(ov.Id, 0, ov.Id, column, item), true
}

func (this *Tree) vtToggle(view int64) bool {
	ov, _, _, itemID, ok := this.vtLocate(view)
	if !ok || !this.IsEnabled() || this.checkColumn == nil {
		return false
	}
	if item := this.display.GetWidget(itemID); item == nil || item.IsDisposed() {
		return false
	}
	this.outlineView_setObjectValue_forTableColumn_byItem_(ov.Id, 0, ov.Id, 0, this.checkColumn.Id, itemID)
	return true
}

func (this *Tree) vtToolTip(view int64) string {
	ov, _, column, item, ok := this.vtLocate(view)
	cell := this.dataCell
	if !ok || this.toolTipText != "" || cell == nil {
		return ""
	}
	value := this.outlineView_objectValueForTableColumn_byItem_(ov.Id, 0, ov.Id, column, item)
	cell.SetObjectValue(cocoa.NewidOverload1(value))
	this.outlineView_willDisplayCell_forTableColumn_item_(ov.Id, 0, ov.Id, cell.Id, column, item)
	if cell.CellSize().Width <= cocoa.NewNSViewOverload1(view).Bounds().Width {
		return ""
	}
	return cocoa.NewNSStringOverload1(value).GetString()
}

func (this *Tree) vtToggleSelected() {
	if this.IsDisposed() || (this.style&CHECK) == 0 {
		return
	}
	ov := castcocoaNSViewTococoaNSOutlineView(this.View)
	selection := ov.SelectedRowIndexes()
	count := int32(selection.Count())
	indices := make([]int64, count)
	selection.GetIndexes(indices, int64(count), int64(0))
	for _, i := range indices {
		this.outlineView_setObjectValue_forTableColumn_byItem_(ov.Id, 0, ov.Id, 0, this.checkColumn.Id, ov.ItemAtRow(i).Id)
	}
}
