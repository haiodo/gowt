package swt

import "github.com/haiodo/gowt/internal/gtk"

const portalSettingsTimeoutMsec = 2000

// FollowSystemTheme makes the display track org.freedesktop.appearance color-scheme from
// xdg-desktop-portal: the value is applied as gtk-application-prefer-dark-theme, DisplayIsSystemDarkTheme
// reflects it and every change fires SWT.Settings. Without a portal nothing changes.
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
		if ns == "org.freedesktop.appearance" && key == "color-scheme" {
			child := gtk.OSG_variant_get_child_value(a[3], 2)
			this.applyColorScheme(portalUint32(child))
			gtk.OSG_variant_unref(child)
		}
		return 0
	}, 5)
	gtk.OSG_signal_connect(proxy, gtk.ConverterJavaStringToCString("g-signal"), cb.GetAddress(), 0)

	args := gtk.OSG_variant_newOverload2(gtk.ConverterJavaStringToCString("(ss)"),
		gtk.ConverterJavaStringToCString("org.freedesktop.appearance"), gtk.ConverterJavaStringToCString("color-scheme"))
	// ReadOne is Settings v2; older portals only have Read.
	res := gtk.OSG_dbus_proxy_call_sync(proxy, gtk.ConverterJavaStringToCString("ReadOne"), args, gtk.OSG_DBUS_CALL_FLAGS_NONE, portalSettingsTimeoutMsec, 0, gerr)
	if res == 0 {
		DisplayExtractFreeGError(gerr[0])
		gerr[0] = 0
		args = gtk.OSG_variant_newOverload2(gtk.ConverterJavaStringToCString("(ss)"),
			gtk.ConverterJavaStringToCString("org.freedesktop.appearance"), gtk.ConverterJavaStringToCString("color-scheme"))
		res = gtk.OSG_dbus_proxy_call_sync(proxy, gtk.ConverterJavaStringToCString("Read"), args, gtk.OSG_DBUS_CALL_FLAGS_NONE, portalSettingsTimeoutMsec, 0, gerr)
		if res == 0 {
			DisplayExtractFreeGError(gerr[0])
			return
		}
	}
	child := gtk.OSG_variant_get_child_value(res, 0)
	scheme := portalUint32(child)
	gtk.OSG_variant_unref(child)
	gtk.OSG_variant_unref(res)
	this.applyColorScheme(scheme)
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
