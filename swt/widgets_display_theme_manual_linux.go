package swt

import (
	"fmt"

	"github.com/haiodo/gowt/internal/gtk"
)

const portalSettingsTimeoutMsec = 2000

// FollowSystemTheme is opt-in for plain swt users (gowt.Run calls it). It makes the display track org.freedesktop.appearance color-scheme from
// xdg-desktop-portal: the value is applied as gtk-application-prefer-dark-theme, DisplayIsSystemDarkTheme
// reflects it and every change fires SWT.Settings. The portal accent-color goes to the @gowt_accent named color
// (see gtkres/swt_functional_gtk_3_20.css). Without a portal nothing changes.
func (this *Display) FollowSystemTheme() {
	var gerr []int64 = make([]int64, 1)
	proxy := gtk.OSG_dbus_proxy_new_for_bus_sync(gtk.OSG_BUS_TYPE_SESSION, gtk.OSG_DBUS_PROXY_FLAGS_DO_NOT_LOAD_PROPERTIES, 0,
		gtk.ConverterJavaStringToCString("org.freedesktop.portal.Desktop"),
		gtk.ConverterJavaStringToCString("/org/freedesktop/portal/desktop"),
		gtk.ConverterJavaStringToCString("org.freedesktop.portal.Settings"), 0, gerr)
	if proxy == 0 {
		DisplayExtractFreeGError(gerr[0])
		return
	}
	// The proxy and its callback live as long as the process; one display per process is the supported case.
	cb := NewCallbackFn(func(a []int64) int64 {
		if gtk.ConverterCCharPtrToJavaString(a[2], false) != "SettingChanged" {
			return 0
		}
		ns := portalChildString(a[3], 0)
		key := portalChildString(a[3], 1)
		if ns != "org.freedesktop.appearance" {
			return 0
		}
		child := gtk.OSG_variant_get_child_value(a[3], 2)
		defer gtk.OSG_variant_unref(child)
		switch key {
		case "color-scheme":
			this.applyColorScheme(portalUint32(child))
		case "accent-color":
			this.applyAccent(portalDoubles(child))
		}
		return 0
	}, 5)
	gtk.OSG_signal_connect(proxy, gtk.ConverterJavaStringToCString("g-signal"), cb.GetAddress(), 0)

	if res := portalRead(proxy, "color-scheme"); res != 0 {
		child := gtk.OSG_variant_get_child_value(res, 0)
		this.applyColorScheme(portalUint32(child))
		gtk.OSG_variant_unref(child)
		gtk.OSG_variant_unref(res)
	}
	if res := portalRead(proxy, "accent-color"); res != 0 {
		child := gtk.OSG_variant_get_child_value(res, 0)
		this.applyAccent(portalDoubles(child))
		gtk.OSG_variant_unref(child)
		gtk.OSG_variant_unref(res)
	}
}

// portalRead returns the (v) reply of the appearance namespace key, 0 when the portal lacks it.
func portalRead(proxy int64, key string) int64 {
	var gerr []int64 = make([]int64, 1)
	for _, method := range []string{"ReadOne", "Read"} { // ReadOne is Settings v2; older portals only have Read
		args := gtk.OSG_variant_newOverload2(gtk.ConverterJavaStringToCString("(ss)"),
			gtk.ConverterJavaStringToCString("org.freedesktop.appearance"), gtk.ConverterJavaStringToCString(key))
		res := gtk.OSG_dbus_proxy_call_sync(proxy, gtk.ConverterJavaStringToCString(method), args, gtk.OSG_DBUS_CALL_FLAGS_NONE, portalSettingsTimeoutMsec, 0, gerr)
		if res != 0 {
			return res
		}
		DisplayExtractFreeGError(gerr[0])
		gerr[0] = 0
	}
	return 0
}

// accentProvider holds the one CSS provider that defines @gowt_accent; it outranks the application CSS
// that only defines the theme fallback.
var accentProvider int64

// applyAccent takes the portal accent-color (r, g, b in 0..1; out of range means no accent: the theme's selection color stays).
func (this *Display) applyAccent(rgb []float64) {
	if this.IsDisposed() {
		return
	}
	css := ""
	if len(rgb) == 3 && rgb[0] >= 0 && rgb[0] <= 1 && rgb[1] >= 0 && rgb[1] <= 1 && rgb[2] >= 0 && rgb[2] <= 1 {
		fg := "#ffffff"
		if 0.2126*rgb[0]+0.7152*rgb[1]+0.0722*rgb[2] > 0.6 {
			fg = "#000000"
		}
		css = fmt.Sprintf("@define-color gowt_accent #%02x%02x%02x;\n@define-color gowt_accent_fg %s;\n",
			int(rgb[0]*255+0.5), int(rgb[1]*255+0.5), int(rgb[2]*255+0.5), fg)
	}
	if accentProvider == 0 {
		screen := gtk.GDKGdk_screen_get_default()
		if screen == 0 || css == "" {
			return
		}
		accentProvider = gtk.GTKGtk_css_provider_new()
		gtk.GTK3Gtk_style_context_add_provider_for_screen(screen, accentProvider, gtk.GTKGTK_STYLE_PROVIDER_PRIORITY_USER)
	}
	gtk.GTK3Gtk_css_provider_load_from_data(accentProvider, gtk.ConverterWcsToMbcs(css, true), int64(-1), nil)
	this.settingsChanged = true
}

// applyColorScheme takes the portal value: 1 dark, 2 light, 0 no preference (the GTK theme stays as it is).
func (this *Display) applyColorScheme(scheme uint32) {
	if scheme != 1 && scheme != 2 {
		return
	}
	dark := scheme == 1
	if dark == DisplayThemeDark || this.IsDisposed() {
		return
	}
	gtk.OSSetTheme(dark)
	DisplayThemeDark = this.CheckAndSetThemeDetails(DisplayThemeName)
	this.settingsChanged = true
}

func portalChildString(tuple int64, i int32) string {
	child := gtk.OSG_variant_get_child_value(tuple, i)
	defer gtk.OSG_variant_unref(child)
	return gtk.ConverterCCharPtrToJavaString(gtk.OSG_variant_get_string(child, nil), false)
}

// portalUint32 unwraps the 'v' layers (ReadOne gives one, Read two) and returns the uint32 inside.
func portalUint32(v int64) uint32 {
	for gtk.ConverterCCharPtrToJavaString(gtk.OSG_variant_get_type_string(v), false) == "v" {
		inner := gtk.OSG_variant_get_variant(v)
		defer gtk.OSG_variant_unref(inner)
		v = inner
	}
	return uint32(gtk.OSG_variant_get_uint32(v))
}

// portalDoubles unwraps the 'v' layers and returns the doubles of the tuple inside (accent-color is (ddd)).
func portalDoubles(v int64) []float64 {
	for gtk.ConverterCCharPtrToJavaString(gtk.OSG_variant_get_type_string(v), false) == "v" {
		inner := gtk.OSG_variant_get_variant(v)
		defer gtk.OSG_variant_unref(inner)
		v = inner
	}
	n := int32(gtk.OSG_variant_n_children(v))
	out := make([]float64, 0, n)
	for i := int32(0); i < n; i++ {
		c := gtk.OSG_variant_get_child_value(v, i)
		out = append(out, gtk.OSG_variant_get_double(c))
		gtk.OSG_variant_unref(c)
	}
	return out
}
