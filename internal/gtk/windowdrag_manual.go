//go:build linux

package gtk

var (
	gtkWindowBeginMoveDrag = lz[func(uintptr, int32, int32, int32, uint32)]{sym: "gtk_window_begin_move_drag"}
	gtkGetCurrentEventTime = lz[func() uint32]{sym: "gtk_get_current_event_time"}
)

// BeginMoveDrag asks the window manager to move the toplevel of widget with the pointer, as a title bar drag does (GTK 3).
// button, rootX and rootY are those of the press that started the drag.
func BeginMoveDrag(widget int64, button, rootX, rootY int32) {
	win := GTK3Gtk_widget_get_toplevel(widget)
	if win == 0 {
		return
	}
	gtkWindowBeginMoveDrag.get()(uintptr(win), button, rootX, rootY, gtkGetCurrentEventTime.get()())
}
