//go:build darwin

// Hand-written: Liquid Glass (macOS 26), NSVisualEffectView backdrops, window chrome and the
// compatibility opt-out. Every AppKit class and selector newer than the deployment target
// (13.0) is looked up at run time and skipped when absent.
package cocoa

import (
	"github.com/ebitengine/purego"
)

const (
	BackdropNone        = 0
	BackdropTranslucent = 1
	BackdropGlass       = 2
)

const (
	viewWidthHeightSizable = 2 | 16
	styleMaskFullSize      = 1 << 15
	bezelStyleGlass        = 16
	materialUnderWindowBG  = 21
)

// NSWindowBelow is -1 as an NSInteger word.
var windowBelowArg = ^uintptr(0)

var msgF64 func(self, sel uintptr, v float64)

// backdropID marks the view InstallBackdrop adds, so a later call can find and replace it.
const backdropID = "gowt.backdrop"

func respondsTo(obj uintptr, selector string) bool {
	return obj != 0 && msg(obj, "respondsToSelector:", sel(selector)) != 0
}

// GlassAvailable reports whether NSGlassEffectView exists (macOS 26 or newer).
func GlassAvailable() bool { return class("NSGlassEffectView") != 0 }

// InstallBackdrop puts a material view behind the content of view, replacing an earlier one.
// kind is BackdropNone, BackdropTranslucent or BackdropGlass; Glass falls back to Translucent
// before macOS 26. With window set, the NSWindow is made non-opaque while a material is shown.
func InstallBackdrop(view *NSView, kind int, window bool) {
	v := uintptr(view.Id)
	subs := msg(v, "subviews")
	for i := msg(subs, "count"); i > 0; i-- {
		s := msg(subs, "objectAtIndex:", i-1)
		if goString(msg(s, "identifier")) == backdropID {
			msg(s, "removeFromSuperview")
		}
	}
	if window {
		if w := msg(v, "window"); w != 0 {
			setWindowOpaque(w, kind == BackdropNone)
		}
	}
	if kind == BackdropNone {
		return
	}
	var bd uintptr
	if kind == BackdropGlass && GlassAvailable() {
		bd = msg(msg(class("NSGlassEffectView"), "alloc"), "init")
	} else {
		bd = msg(msg(class("NSVisualEffectView"), "alloc"), "init")
		msg(bd, "setMaterial:", materialUnderWindowBG)
		msg(bd, "setBlendingMode:", 0)
		msg(bd, "setState:", 1)
	}
	msg(bd, "setIdentifier:", nsString(backdropID))
	msg(bd, "setAutoresizingMask:", viewWidthHeightSizable)
	msgRectOnly(bd, sel("setFrame:"), view.Bounds())
	msg(v, "addSubview:positioned:relativeTo:", bd, windowBelowArg, 0)
	msg(bd, "release")
}

func setWindowOpaque(w uintptr, opaque bool) {
	if opaque {
		msg(w, "setOpaque:", 1)
		msg(w, "setBackgroundColor:", msg(class("NSColor"), "windowBackgroundColor"))
		return
	}
	msg(w, "setOpaque:", 0)
	msg(w, "setBackgroundColor:", msg(class("NSColor"), "clearColor"))
}

// SetFullSizeContent lets the content run under the title bar and makes the bar transparent.
func SetFullSizeContent(view *NSView, on bool) {
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	mask := msg(w, "styleMask")
	if on {
		mask |= styleMaskFullSize
	} else {
		mask &^= styleMaskFullSize
	}
	msg(w, "setStyleMask:", mask)
	b := uintptr(0)
	if on {
		b = 1
	}
	msg(w, "setTitlebarAppearsTransparent:", b)
}

// SetGlassButton gives an NSButton the macOS 26 glass bezel; older systems keep the current one.
func SetGlassButton(view *NSView) {
	v := uintptr(view.Id)
	if !GlassAvailable() || !respondsTo(v, "setBezelStyle:") {
		return
	}
	msg(v, "setBezelStyle:", bezelStyleGlass)
}

// SetGlassCornerRadius rounds the glass view InstallBackdrop added to view; no-op without one.
func SetGlassCornerRadius(view *NSView, r float64) {
	objcInit()
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	subs := msg(uintptr(view.Id), "subviews")
	for i := msg(subs, "count"); i > 0; i-- {
		s := msg(subs, "objectAtIndex:", i-1)
		if goString(msg(s, "identifier")) == backdropID && respondsTo(s, "setCornerRadius:") {
			msgF64(s, sel("setCornerRadius:"), r)
		}
	}
}

// RequireCompatibleLook asks AppKit to keep the pre-26 look (UIDesignRequiresCompatibility).
// Call before NSApplication starts. The key is an Info.plist key; the NSUserDefaults
// registration and the main bundle's info dictionary are the two places a bundle-less binary can set it.
func RequireCompatibleLook() {
	const key = "UIDesignRequiresCompatibility"
	yes := msg(class("NSNumber"), "numberWithBool:", 1)
	dict := msg(class("NSDictionary"), "dictionaryWithObject:forKey:", yes, nsString(key))
	msg(msg(class("NSUserDefaults"), "standardUserDefaults"), "registerDefaults:", dict)
	info := msg(msg(class("NSBundle"), "mainBundle"), "infoDictionary")
	if respondsTo(info, "setObject:forKey:") {
		msg(info, "setObject:forKey:", yes, nsString(key))
	}
}

// CompatibleLookKey reads the key back from the main bundle (for tests and the demo).
func CompatibleLookKey() bool {
	v := msg(msg(class("NSBundle"), "mainBundle"), "objectForInfoDictionaryKey:", nsString("UIDesignRequiresCompatibility"))
	return v != 0 && msg(v, "boolValue")&0xff != 0
}
