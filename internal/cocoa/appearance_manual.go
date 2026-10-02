//go:build darwin

package cocoa

import (
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
			appearanceChanged()
		}), "v@:@@@^v"},
	})
	obs := msg(msg(cls, "alloc"), "init")
	msg(msg(class("NSApplication"), "sharedApplication"), "addObserver:forKeyPath:options:context:", obs, nsString("effectiveAppearance"), 0, 0)
}

// EffectiveAppearanceDark reports whether NSApp.effectiveAppearance resolves to Dark Aqua; unlike the
// AppleInterfaceStyle default it is already updated when the KVO callback runs.
func EffectiveAppearanceDark() bool {
	names := [2]uintptr{nsString("NSAppearanceNameAqua"), nsString("NSAppearanceNameDarkAqua")}
	arr := msg(class("NSArray"), "arrayWithObjects:count:", uintptr(unsafe.Pointer(&names[0])), 2)
	ap := msg(msg(class("NSApplication"), "sharedApplication"), "effectiveAppearance")
	best := msg(ap, "bestMatchFromAppearancesWithNames:", arr)
	runtime.KeepAlive(names)
	return goString(best) == "NSAppearanceNameDarkAqua"
}
