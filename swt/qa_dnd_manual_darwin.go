package swt

// qaWire: the pasteboard of a real drag holds the native form; here it is td.Data.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	return func() {}, qaSetData(ds, td)
}

// qaNoDragImage: the Tree and Table effects pass NSApp.currentEvent, none in a QA drag, to
// dragImageForRowsWithIndexes; a QA drag shows no image, so DragStart runs without them. The event is
// non-nil here on macOS 26 even without a drag session, and a stale one made AppKit throw
// NSImageCacheException on the zero-size drag image of a row, so the effect always goes.
func qaNoDragImage(ds *DragSource) func() {
	e := ds.dragEffect
	if e == nil {
		return func() {}
	}
	switch any(e.impl).(type) {
	case *TreeDragSourceEffect, *TableDragSourceEffect:
		ds.dragEffect = nil
		return func() { ds.dragEffect = e }
	}
	return func() {}
}
