//go:build linux

package gtk

import (
	"os"
	"strings"
	"unsafe"
)

// GList/GSList are plain structs {data, next[, prev]} (GLib docs): the accessors read them directly.
func OSG_list_data(l int64) int64     { return peek64(l) }
func OSG_list_next(l int64) int64     { return peek64(l + 8) }
func OSG_list_previous(l int64) int64 { return peek64(l + 16) }
func OSG_slist_data(l int64) int64    { return peek64(l) }
func OSG_slist_next(l int64) int64    { return peek64(l + 8) }

// GType fundamentals are G_TYPE_MAKE_FUNDAMENTAL(n) = n << 2.
func OSG_TYPE_BOOLEAN() int64 { return 5 << 2 }
func OSG_TYPE_INT() int64     { return 6 << 2 }
func OSG_TYPE_LONG() int64    { return 8 << 2 }
func OSG_TYPE_STRING() int64  { return 16 << 2 }

// A GObject starts with a GTypeInstance whose first word is the class; a GTypeClass starts with its GType.
func OSG_OBJECT_GET_CLASS(obj int64) int64    { return peek64(obj) }
func OSG_OBJECT_TYPE(obj int64) int64         { return peek64(peek64(obj)) }
func GTKGTK_WIDGET_GET_CLASS(obj int64) int64 { return peek64(obj) }

var gTypeName = lz[func(uintptr) uintptr]{sym: "g_type_name"}

func OSG_OBJECT_TYPE_NAME(obj int64) int64 {
	return int64(gTypeName.get()(uintptr(OSG_OBJECT_TYPE(obj))))
}

var gObjectClassConstructor = unsafe.Offsetof(GObjectClass{}.Constructor)

func OSG_OBJECT_CLASS_CONSTRUCTOR(klass int64) int64 {
	return peek64(klass + int64(gObjectClassConstructor))
}
func OSG_OBJECT_CLASS_SET_CONSTRUCTOR(klass, fn int64) {
	poke64(klass+int64(gObjectClassConstructor), fn)
}

var gVariantGetDouble = lz[func(uintptr) float64]{sym: "g_variant_get_double"}

func OSG_variant_get_double(v int64) float64 { return gVariantGetDouble.get()(uintptr(v)) }

// GVariantType pointers are the type string itself; the strings live for the process.
func variantType(s string) int64 {
	p := OSG_malloc(int64(len(s) + 1))
	copy(unsafe.Slice((*byte)(at(p)), len(s)+1), s+"\x00")
	return p
}

var (
	OSG_VARIANT_TYPE_BOOLEAN      = variantType("b")
	OSG_VARIANT_TYPE_IN32         = variantType("i")
	OSG_VARIANT_TYPE_STRING       = variantType("s")
	OSG_VARIANT_TYPE_STRING_ARRAY = variantType("as")
	OSG_VARIANT_TYPE_TUPLE        = variantType("r")
)

const OSDBUS_TYPE_STRING_ARRAY = "as"

var gSignalConnectData = lz[func(uintptr, unsafe.Pointer, uintptr, uintptr, uintptr, int32) uintptr]{sym: "g_signal_connect_data"}

// OSG_signal_connect is the g_signal_connect macro.
func OSG_signal_connect(instance int64, signal []int8, handler, data int64) int32 {
	return int32(gSignalConnectData.get()(uintptr(instance), sp(signal), uintptr(handler), uintptr(data), 0, 0))
}

// utf8Step returns the byte length of the UTF-8 sequence starting with lead byte b.
func utf8Step(b byte) int64 {
	switch {
	case b < 0x80:
		return 1
	case b < 0xe0:
		return 2
	case b < 0xf0:
		return 3
	}
	return 4
}

func utf16Units(b byte) int64 {
	if b >= 0xf0 {
		return 2
	}
	return 1
}

// OSG_utf8_offset_to_utf16_offset converts a character offset into a UTF-8 string to UTF-16 units.
func OSG_utf8_offset_to_utf16_offset(str, offset int64) int64 {
	var units int64
	for p := str; offset > 0; offset-- {
		b := *(*byte)(at(p))
		if b == 0 {
			break
		}
		units += utf16Units(b)
		p += utf8Step(b)
	}
	return units
}

// OSG_utf16_offset_to_utf8_offset converts UTF-16 units into a character offset.
func OSG_utf16_offset_to_utf8_offset(str, offset int64) int64 {
	var chars int64
	for p := str; offset > 0; chars++ {
		b := *(*byte)(at(p))
		if b == 0 {
			break
		}
		offset -= utf16Units(b)
		p += utf8Step(b)
	}
	return chars
}

// OSG_utf16_offset_to_pointer is g_utf8_offset_to_pointer counting UTF-16 units.
func OSG_utf16_offset_to_pointer(str, offset int64) int64 {
	p := str
	for offset > 0 {
		b := *(*byte)(at(p))
		if b == 0 {
			break
		}
		offset -= utf16Units(b)
		p += utf8Step(b)
	}
	return p
}

// OSG_utf16_pointer_to_offset is g_utf8_pointer_to_offset counting UTF-16 units.
func OSG_utf16_pointer_to_offset(str, pos int64) int64 {
	var units int64
	for p := str; p < pos; {
		b := *(*byte)(at(p))
		if b == 0 {
			break
		}
		units += utf16Units(b)
		p += utf8Step(b)
	}
	return units
}

// OSG_utf16_strlen is g_utf8_strlen (max in bytes, -1 for NUL-terminated) counting UTF-16 units.
func OSG_utf16_strlen(str, max int64) int64 {
	var units int64
	for p := str; max < 0 || p < str+max; {
		b := *(*byte)(at(p))
		if b == 0 {
			break
		}
		units += utf16Units(b)
		p += utf8Step(b)
	}
	return units
}

func OSVERSION(major, minor, micro int32) int32 { return major<<16 | minor<<8 | micro }

// PANGO_PIXELS rounds Pango units to pixels.
func OSPANGO_PIXELS(u int32) int32 { return (u + 512) >> 10 }

const OSBIG_ENDIAN = false

// SWT's environment switches.
const (
	OSSWT_LIB_VERSIONS            = "SWT_LIB_VERSIONS"
	OSSWT_DEBUG                   = false
	OSSWT_MENU_LOCATION_DEBUGGING = false
)

var OSSWT_PADDED_MENU_ITEMS = os.Getenv("SWT_PADDED_MENU_ITEMS") != ""
var OSGTK_THEME_SET = os.Getenv("GTK_THEME") != ""
var OSGTK_THEME_SET_NAME = os.Getenv("GTK_THEME")
var OSGTK_OVERLAY_SCROLLING_DISABLED = os.Getenv("GTK_OVERLAY_SCROLLING") == "0"
var OSIsGNOME = strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME")

func OSGetEnvironmentalVariable(name string) string { return os.Getenv(name) }

// GTK_VERSION is the runtime GTK version in OSVERSION encoding; this binding drives GTK 3.
var GTKGTK_VERSION = OSVERSION(
	int32(gtkVersionPart("gtk_get_major_version")),
	int32(gtkVersionPart("gtk_get_minor_version")),
	int32(gtkVersionPart("gtk_get_micro_version")))

const GTKGTK4 = false

func gtkVersionPart(sym string) uint32 {
	f := lz[func() uint32]{sym: sym}
	return f.get()()
}
