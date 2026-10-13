//go:build darwin

package cocoa

// BeginWindowDrag starts a native window move from the mouse-down event being dispatched; it
// returns when the button is released. It reports false outside a mouse-down.
func BeginWindowDrag(view *NSView) bool {
	w := msg(uintptr(view.Id), "window")
	ev := msg(msg(class("NSApplication"), "sharedApplication"), "currentEvent")
	if w == 0 || ev == 0 {
		return false
	}
	msg(w, "performWindowDragWithEvent:", ev)
	return true
}

// TitleBarDoubleClick does what a double click on the title bar does by the user's setting
// (AppleActionOnDoubleClick): zoom, minimize or nothing.
func TitleBarDoubleClick(view *NSView) {
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	defaults := msg(class("NSUserDefaults"), "standardUserDefaults")
	v := msg(defaults, "stringForKey:", nsString("AppleActionOnDoubleClick"))
	action := "Maximize"
	if v != 0 {
		action = goString(v)
	}
	switch action {
	case "Minimize":
		msg(w, "performMiniaturize:", 0)
	case "None":
	default:
		msg(w, "performZoom:", 0)
	}
}
