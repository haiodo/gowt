package swt

import (
	"log"
	"os"

	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt/internal/cocoa"
)

// GOWT_VIEW_TABLE=1 switches Table and Tree from cell-based to view-based NSTableView/NSOutlineView (docs/view-based-table.md).
// Off by default: the mode is not yet verified against tests/expected.txt.
var viewTableEnabled = os.Getenv("GOWT_VIEW_TABLE") == "1"

// Cell views draw through the table's shared data cell (the same EraseItem/MeasureItem/PaintItem flow as drawInteriorWithFrame).
var (
	vtCellClass  int64
	vtCheckClass int64
	vtDrawing    = map[int64]bool{} // cell views inside drawRect: invalidating them from there would loop
	vtViewCalls  int                // viewForTableColumn calls that reached vtMakeView; read by the GUI test
)

func init() {
	if viewTableEnabled {
		if !cocoa.OnClassRegistered(vtClassRegistered) {
			log.Print("gowt: GOWT_VIEW_TABLE=1 ignored: the class registration hook came too late")
		}
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

func vtTableViewFor(a []int64) int64 {
	t := vtAsTable(DisplayGetWidget(a[0]))
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
	cocoa.OSObjc_registerClassPair(cls)
	chk := cocoa.OSObjc_allocateClassPair(cls, "SWTTableCheckView", 0)
	cocoa.OSClass_addMethod(chk, cocoa.OSSel_mouseDown_, cocoa.NewCallbackN(3, vtCheckMouseDown), "v@:@")
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

// vtDrawContext draws with the table's coordinates: the cell view's own origin is shifted back to the table's.
func vtDrawContext(table *cocoa.NSView, view *cocoa.NSView, draw func()) {
	origin := table.ConvertRect_fromView_(view.Bounds(), view)
	ctx := cocoa.NSGraphicsContextCurrentContext()
	ctx.SaveGraphicsState()
	transform := cocoa.NSAffineTransformTransform()
	transform.TranslateXBy(-origin.X, -origin.Y)
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
	cell.SetHighlighted(tv.IsRowSelected(row))
	this.tableView_willDisplayCell_forTableColumn_row_(tv.Id, 0, tv.Id, cell.Id, column, row)
	frame := tv.FrameOfCellAtColumn(col, row)
	vtDrawContext(this.View, v, func() { cell.DrawInteriorWithFrame(frame, this.View) })
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
	vtDrawContext(this.View, v, func() { cell.DrawInteriorWithFrame(frame, this.View) })
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
