package gowt

import "github.com/haiodo/gowt/internal/gtk"

const (
	systemQuitAvailable = false // GTK 3 without GtkApplication has no session end signal
	dialogDismissable   = true
)

func systemQuit(*Window, bool) bool { return false }

// dismissDialog answers the modal dialog No, which makes gtk_dialog_run return.
func dismissDialog(*Window) {
	list := gtk.GTKGtk_window_list_toplevels()
	for i := int32(0); ; i++ {
		h := gtk.OSG_list_nth_data(list, i)
		if h == 0 {
			return
		}
		if gtk.GTKGtk_widget_get_visible(h) && gtk.GTKGtk_window_get_modal(h) {
			gtk.OSG_signal_emit_by_nameOverload1(h, gtk.OSResponse, -9) // GTK_RESPONSE_NO
			return
		}
	}
}
