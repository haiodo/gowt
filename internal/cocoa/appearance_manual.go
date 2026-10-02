//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

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
