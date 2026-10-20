//go:build linux

package gtk

import "unsafe"

const (
	gdkScroll     = 31
	gdkScrollUp   = 0
	gdkScrollDown = 1
)

// PostWheel queues one notch of the mouse wheel over widget's window, down when down is set,
// as the X server delivers it (GTK 3). Tests use it; a test cannot move the real pointer.
func PostWheel(widget int64, down bool) {
	win := GTK3Gtk_widget_get_window(widget)
	if win == 0 {
		return
	}
	ev := GDKGdk_event_new(gdkScroll)
	// GdkEventScroll on LP64: window 8, time 20, x 24, y 32, direction 44 (checked with offsetof).
	p := at(ev)
	*(*int64)(unsafe.Add(p, 8)) = OSG_object_ref(win) // gdk_event_free unrefs it
	*(*uint32)(unsafe.Add(p, 20)) = gtkGetCurrentEventTime.get()()
	*(*float64)(unsafe.Add(p, 24)) = 10
	*(*float64)(unsafe.Add(p, 32)) = 10
	dir := uint32(gdkScrollUp)
	if down {
		dir = gdkScrollDown
	}
	*(*uint32)(unsafe.Add(p, 44)) = dir
	GDKGdk_event_set_device(ev, GDKGdk_seat_get_pointer(GDKGdk_display_get_default_seat(GDKGdk_display_get_default())))
	GDKGdk_event_put(ev)
	GDKGdk_event_free(ev)
}
