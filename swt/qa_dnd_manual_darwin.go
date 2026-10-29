package swt

// qaWire: the pasteboard of a real drag holds the native form; here it is td.Data.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	return func() {}, qaSetData(ds, td)
}

// qaNoDragImage: the Tree and Table effects pass NSApp.currentEvent to dragImageForRowsWithIndexes,
// and on macOS 26 a stale one threw NSImageCacheException on the zero-size image: they always go.
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
