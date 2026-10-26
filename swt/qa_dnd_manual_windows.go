package swt

// qaWire: a Windows Transfer reads from the source's IDataObject, whose GetData sends DragSetData.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	ds.CreateCOMInterfaces()
	td.PIDataObject = ds.iDataObject.GetAddress()
	return func() {
		td.PIDataObject = 0
		ds.ReleaseCOMInterfaces()
	}, true
}
