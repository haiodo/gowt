package swt

import "github.com/haiodo/gowt/internal/cocoa"

// qaWire: the pasteboard of a real drag holds the native form; here it is td.Data.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	return func() {}, qaSetData(ds, td)
}

// qaNoDragImage: the Tree and Table effects pass NSApp.currentEvent, nil in a QA drag, to
// dragImageForRowsWithIndexes; a QA drag shows no image, so DragStart runs without them.
func qaNoDragImage(ds *DragSource) func() {
	e := ds.dragEffect
	if e == nil || cocoa.NSApplicationSharedApplication().CurrentEvent() != nil {
		return func() {}
	}
	switch any(e.impl).(type) {
	case *TreeDragSourceEffect, *TableDragSourceEffect:
		ds.dragEffect = nil
		return func() { ds.dragEffect = e }
	}
	return func() {}
}
