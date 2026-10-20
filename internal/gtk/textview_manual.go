//go:build linux

package gtk

import "unsafe"

var (
	gtkCairoShouldDrawWindow   = lz[func(uintptr, uintptr) int32]{sym: "gtk_cairo_should_draw_window"}
	gtkCairoTransformToWindow  = lz[func(uintptr, uintptr, uintptr)]{sym: "gtk_cairo_transform_to_window"}
	gtkStyleContextLookupColor = lz[func(uintptr, unsafe.Pointer, *GdkRGBA) int32]{sym: "gtk_style_context_lookup_color"}
)

// PlaceholderColor is the theme's placeholder_text_color, else the 50% grey a GtkEntry
// placeholder measured in Adwaita light and dark.
func PlaceholderColor(widget int64) GdkRGBA {
	var c GdkRGBA
	if gtkStyleContextLookupColor.get()(uintptr(GTKGtk_widget_get_style_context(widget)), sp(cs("placeholder_text_color")), &c) == 0 {
		return GdkRGBA{Red: 0.5, Green: 0.5, Blue: 0.5, Alpha: 1}
	}
	return c
}

const gConnectAfter = 1

// DrawOverTextView calls draw after the GtkTextView has painted its text window, cr in that
// window's coordinates (GTK 3). The returned function disconnects the handler.
func DrawOverTextView(view int64, draw func(cr int64)) (disconnect func()) {
	cb := NewCallbackN(3, func(a []int64) int64 {
		widget, cr := a[0], a[1]
		win := GTK3Gtk_text_view_get_window(widget, GTKGTK_TEXT_WINDOW_TEXT)
		if win == 0 || gtkCairoShouldDrawWindow.get()(uintptr(cr), uintptr(win)) == 0 {
			return 0
		}
		CairoCairo_save(cr)
		gtkCairoTransformToWindow.get()(uintptr(cr), uintptr(widget), uintptr(win))
		draw(cr)
		CairoCairo_restore(cr)
		return 0
	})
	id := gSignalConnectData.get()(uintptr(view), sp(cs("draw")), uintptr(cb), 0, 0, gConnectAfter)
	return func() {
		OSG_signal_handler_disconnect(view, int64(id))
		FreeCallback(cb)
	}
}
