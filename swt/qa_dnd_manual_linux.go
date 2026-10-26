package swt

import "github.com/haiodo/gowt/internal/gtk"

// qaWire: GTK hands the target a copy of the selection data; here it is td.PValue.
func qaWire(ds *DragSource, td *TransferData) (func(), bool) {
	if !qaSetData(ds, td) {
		return nil, false
	}
	return func() {
		if td.PValue != 0 {
			gtk.OSG_free(td.PValue)
			td.PValue = 0
		}
	}, true
}
