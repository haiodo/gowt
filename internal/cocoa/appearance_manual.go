//go:build darwin

package cocoa

import (
	"math"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

var appearanceChanged func()

// ObserveEffectiveAppearance calls fn on the UI thread after NSApp.effectiveAppearance changed. KVO,
// not AppleInterfaceThemeChangedNotification: it fires once the new appearance is applied, so colors read in fn are current.
// The observer lives for the process; a later call only replaces fn.
func ObserveEffectiveAppearance(fn func()) {
	first := appearanceChanged == nil
	appearanceChanged = fn
	if !first {
		return
	}
	cls := newClass("GowtAppearanceObserver", nil, map[string]objcMethod{
		"observeValueForKeyPath:ofObject:change:context:": {purego.NewCallback(func(self, _, keyPath, obj, change, ctx uintptr) {
			SyncCurrentAppearance()
			appearanceChanged()
		}), "v@:@@@^v"},
	})
	obs := msg(msg(cls, "alloc"), "init")
	msg(msg(class("NSApplication"), "sharedApplication"), "addObserver:forKeyPath:options:context:", obs, nsString("effectiveAppearance"), 0, 0)
}

// EffectiveAppearanceDark reports whether NSApp.effectiveAppearance resolves to Dark Aqua; unlike the
// AppleInterfaceStyle default it is already updated when the KVO callback runs.
func EffectiveAppearanceDark() bool {
	return appearanceDark(msg(msg(class("NSApplication"), "sharedApplication"), "effectiveAppearance"))
}

func appearanceDark(ap uintptr) bool {
	names := [2]uintptr{nsString("NSAppearanceNameAqua"), nsString("NSAppearanceNameDarkAqua")}
	arr := msg(class("NSArray"), "arrayWithObjects:count:", uintptr(unsafe.Pointer(&names[0])), 2)
	best := msg(ap, "bestMatchFromAppearancesWithNames:", arr)
	runtime.KeepAlive(names)
	return goString(best) == "NSAppearanceNameDarkAqua"
}

// SyncCurrentAppearance makes dynamic NSColors (windowBackgroundColor, controlTextColor, ...) resolve under
// NSApp.effectiveAppearance; outside a draw pass they otherwise resolve under a stale +currentAppearance,
// so the cached SWT system colors kept the old theme.
func SyncCurrentAppearance() {
	ap := msg(msg(class("NSApplication"), "sharedApplication"), "effectiveAppearance")
	msg(class("NSAppearance"), "setCurrentAppearance:", ap)
}

// SetAppearance forces the appearance of every window of the app (NSApp.appearance): 1 Dark Aqua,
// 2 Aqua, anything else follows the system again. The effectiveAppearance observer fires afterwards.
func SetAppearance(scheme int32) {
	app := msg(class("NSApplication"), "sharedApplication")
	var ap uintptr
	switch scheme {
	case 1:
		ap = msg(class("NSAppearance"), "appearanceNamed:", nsString("NSAppearanceNameDarkAqua"))
	case 2:
		ap = msg(class("NSAppearance"), "appearanceNamed:", nsString("NSAppearanceNameAqua"))
	}
	msg(app, "setAppearance:", ap)
	SyncCurrentAppearance()
}

// SecondaryLabelRGB is secondaryLabelColor as 0..255 sRGB, flattened over windowBackgroundColor as
// GroupedFillRGB is.
func SecondaryLabelRGB() (r, g, b int32) {
	if r, g, b, ok := overWindow(msg(class("NSColor"), "secondaryLabelColor")); ok {
		return r, g, b
	}
	return 128, 128, 128
}

// overWindow resolves the colour c for the app's current appearance and returns it over
// windowBackgroundColor as 0..255 sRGB. SWT paints controls in opaque colours, so a translucent system
// colour is flattened here.
func overWindow(c uintptr) (r, g, b int32, ok bool) {
	appearance := class("NSAppearance")
	old := msg(appearance, "currentAppearance")
	msg(old, "retain")
	msg(appearance, "setCurrentAppearance:", msg(msg(class("NSApplication"), "sharedApplication"), "effectiveAppearance"))
	defer func() {
		msg(appearance, "setCurrentAppearance:", old)
		msg(old, "release")
	}()
	srgb := msg(class("NSColorSpace"), "sRGBColorSpace")
	bg := NewNSColorOverload1(int64(msg(msg(class("NSColor"), "windowBackgroundColor"), "colorUsingColorSpace:", srgb)))
	fill := NewNSColorOverload1(int64(msg(c, "colorUsingColorSpace:", srgb)))
	if bg.Id == 0 || fill.Id == 0 {
		return 0, 0, 0, false
	}
	a := fill.AlphaComponent()
	mix := func(b, f float64) int32 { return int32(math.Round(255 * (b*(1-a) + f*a))) }
	return mix(bg.RedComponent(), fill.RedComponent()), mix(bg.GreenComponent(), fill.GreenComponent()), mix(bg.BlueComponent(), fill.BlueComponent()), true
}
