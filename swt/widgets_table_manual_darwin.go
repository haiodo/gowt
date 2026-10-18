package swt

import "github.com/haiodo/gowt/internal/cocoa"

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
