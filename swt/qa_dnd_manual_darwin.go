package swt

// qaWire: the pasteboard of a real drag holds the native form; here it is td.Data.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	return func() {}, qaSetData(ds, td)
}
