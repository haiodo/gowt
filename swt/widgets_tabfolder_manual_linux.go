package swt

import "github.com/haiodo/gowt/internal/gtk"

var tabPageAllocate *Callback

// hookPage sizes the selected page's control again when GTK allocates its notebook page (port.sh
// adds the call to CreateItem): an unallocated page reports a 1x1 client area to the control.
func (this *TabFolder) hookPage(pageHandle int64) {
	if tabPageAllocate == nil {
		tabPageAllocate = NewCallbackTyped(tabPageAllocated)
	}
	gtk.OSG_signal_connect(pageHandle, gtk.OSSize_allocate, tabPageAllocate.GetAddress(), 0)
}

func tabPageAllocated(page, _, _ uintptr) {
	display := DisplayGetCurrent()
	if display == nil {
		return
	}
	w := display.GetWidget(gtk.GTKGtk_widget_get_parent(int64(page)))
	if w == nil {
		return
	}
	f, ok := w.Impl().(interface{ AsTabFolder() *TabFolder })
	if !ok {
		return
	}
	this := f.AsTabFolder()
	index := gtk.GTKGtk_notebook_get_current_page(this.Handle)
	if index < 0 || int(index) >= len(this.items) || this.items[index] == nil || this.items[index].pageHandle != int64(page) {
		return
	}
	if control := this.items[index].control; control != nil && !control.IsDisposed() {
		control.SetBoundsInPixels(this.impl.getClientAreaInPixels_())
	}
}
