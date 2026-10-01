//go:build linux

package gtk

import "unsafe"

// GtkAdjustment mirrors the values SWT reads and writes through gtk_adjustment_* (the C struct is private).
type GtkAdjustment struct {
	Lower, Upper, Value                       float64
	Step_increment, Page_increment, Page_size float64
}

// cairo_path_t and the header view of cairo_path_data_t (cairo.h; the union is 16 bytes).
type Cairo_path_t struct {
	Status   int32
	Data     int64
	Num_data int32
}

type Cairo_path_data_t struct {
	Type   int32
	Length int32
	_      [8]byte
}

const (
	Cairo_path_tSizeof      = 24
	Cairo_path_data_tSizeof = 16
)

func (r *Cairo_rectangle_int_t) ConvertFromGdkRectangle(g GdkRectangle) {
	r.X, r.Y, r.Width, r.Height = g.X, g.Y, g.Width, g.Height
}

// CAIRO_VERSION_ENCODE from cairo-version.h.
func CairoCAIRO_VERSION_ENCODE(major, minor, micro int32) int32 {
	return major*10000 + minor*100 + micro
}

// GError is {GQuark domain; gint code; gchar *message}.
func OSG_error_get_message(err int64) int64 { return peek64(err + 8) }

var (
	gdkDisplayGetDefaultSeat = lz[func(uintptr) uintptr]{sym: "gdk_display_get_default_seat"}
	gdkSeatGetPointer        = lz[func(uintptr) uintptr]{sym: "gdk_seat_get_pointer"}
)

// GDKGdk_get_pointer is the pointer device of the display's default seat.
func GDKGdk_get_pointer(display int64) int64 {
	return int64(gdkSeatGetPointer.get()(gdkDisplayGetDefaultSeat.get()(uintptr(display))))
}

var (
	gValueInit     = lz[func(unsafe.Pointer, uintptr) uintptr]{sym: "g_value_init"}
	gValueSetFloat = lz[func(unsafe.Pointer, float32)]{sym: "g_value_set_float"}
	gValueUnset    = lz[func(unsafe.Pointer)]{sym: "g_value_unset"}
	gObjectSetProp = lz[func(uintptr, unsafe.Pointer, unsafe.Pointer)]{sym: "g_object_set_property"}
)

// OSG_object_setOverload4 sets a gfloat property; it goes through a GValue because C variadic
// floating-point arguments are not reliable through purego on amd64.
func OSG_object_setOverload4(obj int64, name []int8, value float32, _ int64) {
	var v [24]byte
	gValueInit.get()(unsafe.Pointer(&v), uintptr(14<<2)) // G_TYPE_FLOAT
	gValueSetFloat.get()(unsafe.Pointer(&v), value)
	gObjectSetProp.get()(uintptr(obj), sp(name), unsafe.Pointer(&v))
	gValueUnset.get()(unsafe.Pointer(&v))
}

var (
	imSetClientWindow = lz[func(uintptr, uintptr)]{sym: "gtk_im_context_set_client_window"}
	imClientWindow    = map[int64]int64{}
)

// GTKGtk_im_context_set_client_window keeps a reference on the window an input context was given until it is
// replaced: GtkIMMulticontext reads its previous client window when the next one is set, and a window GDK has
// already destroyed and freed (SwtFixed recreates its window on realize) crashes gdk_window_get_screen. The
// reference of a context that is finalized without a final call leaks one window.
func GTKGtk_im_context_set_client_window(context int64, window int64) {
	if window != 0 {
		gObjectRef.get()(uintptr(window))
	}
	imSetClientWindow.get()(uintptr(context), uintptr(window))
	if old := imClientWindow[context]; old != 0 {
		gObjectUnref.get()(uintptr(old))
	}
	imClientWindow[context] = window
}
