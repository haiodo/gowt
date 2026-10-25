//go:build linux

package view

import (
	"sync"

	"github.com/ebitengine/purego"
)

// GTK 3 entry points beyond what swt binds; swt's display has loaded GTK before attach runs.
var (
	gtkOnce sync.Once

	signalConnect    func(inst uintptr, sig string, handler uintptr, data uintptr, destroy uintptr, flags int32) uint64
	signalDisconnect func(inst uintptr, id uint64)
	objectRef        func(o uintptr) uintptr
	addEvents        func(w uintptr, mask int32)
	scaleFactor      func(w uintptr) int32
	addTick          func(w, cb, data, notify uintptr) uint32
	removeTick       func(w uintptr, id uint32)
	clockTime        func(c uintptr) int64
	clockRefresh     func(c uintptr, base int64, interval, presentation *int64)
	scrollDeltas     func(ev uintptr, dx, dy *float64) int32
	scrollDirection  func(ev uintptr, dir *int32) int32
	scrollIsStop     func(ev uintptr) int32
	eventState       func(ev uintptr, state *uint32) int32
	eventSourceDev   func(ev uintptr) uintptr
	deviceSource     func(d uintptr) int32
	gestureZoomNew   func(w uintptr) uintptr
	zoomDelta        func(g uintptr) float64
	gestureCenter    func(g uintptr, x, y *float64) int32
)

func bindGTK() {
	gtkOnce.Do(func() {
		for fn, name := range map[any]string{
			&signalConnect: "g_signal_connect_data", &signalDisconnect: "g_signal_handler_disconnect",
			&objectRef: "g_object_ref", &addEvents: "gtk_widget_add_events", &scaleFactor: "gtk_widget_get_scale_factor",
			&addTick: "gtk_widget_add_tick_callback", &removeTick: "gtk_widget_remove_tick_callback",
			&clockTime: "gdk_frame_clock_get_frame_time", &clockRefresh: "gdk_frame_clock_get_refresh_info",
			&scrollDeltas: "gdk_event_get_scroll_deltas", &scrollDirection: "gdk_event_get_scroll_direction",
			&scrollIsStop: "gdk_event_is_scroll_stop_event", &eventState: "gdk_event_get_state",
			&eventSourceDev: "gdk_event_get_source_device", &deviceSource: "gdk_device_get_source",
			&gestureZoomNew: "gtk_gesture_zoom_new", &zoomDelta: "gtk_gesture_zoom_get_scale_delta",
			&gestureCenter: "gtk_gesture_get_bounding_box_center",
		} {
			purego.RegisterLibFunc(fn, purego.RTLD_DEFAULT, name)
		}
	})
}

// signal is a handler connected to a GObject, to be disconnected when the view goes.
type signal struct {
	inst uintptr
	id   uint64
}

func connect(inst uintptr, sig string, cb any) signal {
	return signal{inst, signalConnect(inst, sig, purego.NewCallback(cb), 0, 0, 0)}
}

const (
	gdkScrollMask       = 1 << 21
	gdkSmoothScrollMask = 1 << 23
	gdkTouchpadMask     = 1 << 24

	sourceTouchpad = 6

	scrollSmooth = 4
)
