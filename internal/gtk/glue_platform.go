//go:build linux

package gtk

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/internal/jrt"
)

var gdkDisplayGetDefault = lz[func() uintptr]{sym: "gdk_display_get_default"}

func displayTypeName() string {
	d := gdkDisplayGetDefault.get()()
	if d == 0 {
		return ""
	}
	return string(goBytes(OSG_OBJECT_TYPE_NAME(int64(d)), -1))
}

// guessBackend is what GTK would pick for this environment, for the time before a display exists.
func guessBackend() string {
	switch os.Getenv("GDK_BACKEND") {
	case "wayland":
		return "GdkWaylandDisplay"
	case "x11":
		return "GdkX11Display"
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "GdkWaylandDisplay"
	}
	return "GdkX11Display"
}

func backend() string {
	if n := displayTypeName(); n != "" {
		return n
	}
	return guessBackend()
}

func OSIsX11() bool     { return backend() == "GdkX11Display" }
func OSIsWayland() bool { return backend() == "GdkWaylandDisplay" }

var gtkSettingsGetDefault = lz[func() uintptr]{sym: "gtk_settings_get_default"}
var gObjectGetStr = lz[func(uintptr, unsafe.Pointer, unsafe.Pointer, uintptr)]{sym: "g_object_get"}
var gObjectSetInt = lz[func(uintptr, unsafe.Pointer, int32, uintptr)]{sym: "g_object_set"}

// OSGetThemeName is the gtk-theme-name setting.
func OSGetThemeName() string {
	var p int64
	gObjectGetStr.get()(gtkSettingsGetDefault.get()(), sp(cs("gtk-theme-name")), unsafe.Pointer(&p), 0)
	return ConverterCCharPtrToJavaString(p, true)
}

// OSSetTheme sets the dark-theme preference of the default GtkSettings.
func OSSetTheme(dark bool) {
	gObjectSetInt.get()(gtkSettingsGetDefault.get()(), sp(cs("gtk-application-prefer-dark-theme")), b2i(dark), 0)
}

// OSUbuntu_menu_proxy_get reports the Ubuntu global-menu proxy, which this binding never has.
func OSUbuntu_menu_proxy_get() int64 { return 0 }

// GTKGET_FUNCTION_POINTER_gtk_false is the address of gtk_false().
func GTKGET_FUNCTION_POINTER_gtk_false() int64 {
	a, _ := sym("gtk_false")
	return int64(a)
}

var localeconv = lz[func() uintptr]{sym: "localeconv"}

// OSLocaleconv_decimal_point is struct lconv.decimal_point, its first member.
func OSLocaleconv_decimal_point() int64 { return peek64(int64(localeconv.get()())) }

// The GDK lock SWT enters once on the UI thread must be recursive (GDK takes it again from its own
// timeouts), so the default lock functions are replaced with a thread-owned recursive mutex.
var gdkLock struct {
	mu    sync.Mutex
	owner int
	depth int
}

func gdkLockEnter() {
	tid := syscall.Gettid()
	if gdkLock.depth > 0 && gdkLock.owner == tid {
		gdkLock.depth++
		return
	}
	gdkLock.mu.Lock()
	gdkLock.owner, gdkLock.depth = tid, 1
}

func gdkLockLeave() {
	if gdkLock.depth--; gdkLock.depth == 0 {
		gdkLock.owner = 0
		gdkLock.mu.Unlock()
	}
}

var gdkThreadsSetLockFunctions = lz[func(uintptr, uintptr)]{sym: "gdk_threads_set_lock_functions"}

var lockOnce sync.Once

// OSSwt_set_lock_functions may run once per process: GDK refuses a second set.
func OSSwt_set_lock_functions() {
	lockOnce.Do(func() {
		gdkThreadsSetLockFunctions.get()(uintptr(NewCallbackAny(gdkLockEnter)), uintptr(NewCallbackAny(gdkLockLeave)))
	})
}

// Xlib constants (X.h) and event structs (Xlib.h) SWT reads from the raw events GDK passes along.
const (
	OSCurrentTime            = 0
	OSExposureMask           = 1 << 15
	OSFocusIn                = 9
	OSFocusOut               = 10
	OSExpose                 = 12
	OSGraphicsExpose         = 13
	OSNotifyNormal           = 0
	OSNotifyWhileGrabbed     = 3
	OSNotifyAncestor         = 0
	OSNotifyVirtual          = 1
	OSNotifyNonlinear        = 3
	OSNotifyNonlinearVirtual = 4
	OSRevertToParent         = 2
)

type XExposeEvent struct {
	Type       int32
	Serial     int64
	Send_event int32
	Display    int64
	Window     int64
	X, Y       int32
	Width      int32
	Height     int32
	Count      int32
}

type XFocusChangeEvent struct {
	Type       int32
	Serial     int64
	Send_event int32
	Display    int64
	Window     int64
	Mode       int32
	Detail     int32
}

const (
	XExposeEventSizeof      = int64(unsafe.Sizeof(XExposeEvent{}))
	XFocusChangeEventSizeof = int64(unsafe.Sizeof(XFocusChangeEvent{}))
	XEventSizeof            = 24 * 8 // XEvent is a union padded to long pad[24]
)

// Every XEvent starts with type, serial, send_event, display, window (XAnyEvent).
func OSX_EVENT_TYPE(xEvent int64) int32   { return peek32(xEvent) }
func OSX_EVENT_WINDOW(xEvent int64) int64 { return peek64(xEvent + 32) }

// SWT code and the tests ask for the GTK version and the backend through these properties before
// any Display exists (Display sets the backend again from the real display).
func init() {
	b := "x11"
	if OSIsWayland() {
		b = "wayland"
	}
	jrt.SetProperty("org.eclipse.swt.internal.gdk.backend", b)
	jrt.SetProperty("org.eclipse.swt.internal.gtk.version", fmt.Sprintf("%d.%d.%d", gtkVersionPart("gtk_get_major_version"), gtkVersionPart("gtk_get_minor_version"), gtkVersionPart("gtk_get_micro_version")))
}
