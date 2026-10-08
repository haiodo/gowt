//go:build darwin

// Hand-written: Liquid Glass (macOS 26), NSVisualEffectView backdrops, window chrome and the
// compatibility opt-out. Every AppKit class and selector newer than the deployment target
// (13.0) is looked up at run time and skipped when absent.
package cocoa

import (
	"fmt"

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

var (
	setAssociated func(obj, key, value uintptr, policy uint)
	getAssociated func(obj, key uintptr) uintptr
)

// backdropKey is the associated-object key under which a host view retains its backdrop; it
// goes away with the host, and a second install finds and replaces the first.
func backdropKey() uintptr { return sel("gowtBackdrop") }

func initAssociated() {
	objcInit()
	if setAssociated == nil {
		purego.RegisterLibFunc(&setAssociated, purego.RTLD_DEFAULT, "objc_setAssociatedObject")
		purego.RegisterLibFunc(&getAssociated, purego.RTLD_DEFAULT, "objc_getAssociatedObject")
	}
}

func respondsTo(obj uintptr, selector string) bool {
	return obj != 0 && msg(obj, "respondsToSelector:", sel(selector)) != 0
}

// GlassAvailable reports whether NSGlassEffectView exists (macOS 26 or newer).
func GlassAvailable() bool { return class("NSGlassEffectView") != 0 }

// InstallBackdrop puts a material view directly behind view and returns whether it replaced or
// removed an earlier one. kind is BackdropNone, BackdropTranslucent or BackdropGlass; Glass falls
// back to Translucent before macOS 26.
//
// The material is a sibling of view, not a child: SWT inserts every new child at the bottom of its
// parent (Control.setZOrder_ with NSWindowBelow), which would bury a child backdrop under the
// widgets. With window set, view is the shell's content view: the material goes into the window's
// frame view, follows the window size by autoresizing, and the NSWindow is non-opaque while it
// shows and gets the default background back only when a material was removed. Otherwise the
// caller keeps the frame in step with SyncBackdrop.
func InstallBackdrop(view *NSView, kind int, window bool) (had bool) {
	initAssociated()
	v := uintptr(view.Id)
	if old := getAssociated(v, backdropKey()); old != 0 {
		msg(old, "removeFromSuperview")
		setAssociated(v, backdropKey(), 0, 1)
		had = true
	}
	host := msg(v, "superview")
	if w := msg(v, "window"); window && w != 0 && (kind != BackdropNone || had) {
		setWindowOpaque(w, kind == BackdropNone)
	}
	if kind == BackdropNone || host == 0 {
		return had
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
	if window {
		msg(bd, "setAutoresizingMask:", viewWidthHeightSizable)
		msgRectOnly(bd, sel("setFrame:"), NewNSViewOverload1(int64(host)).Bounds())
	} else {
		msgRectOnly(bd, sel("setFrame:"), view.Frame())
	}
	msg(host, "addSubview:positioned:relativeTo:", bd, windowBelowArg, v)
	setAssociated(v, backdropKey(), bd, 1)
	msg(bd, "release")
	return had
}

// SyncBackdrop moves the material added by InstallBackdrop to the frame of view.
func SyncBackdrop(view *NSView) {
	initAssociated()
	if bd := getAssociated(uintptr(view.Id), backdropKey()); bd != 0 {
		msgRectOnly(bd, sel("setFrame:"), view.Frame())
	}
}

// DumpViews describes the view tree of the window of view, from its frame view down, depth levels
// (children listed bottom to top, as AppKit paints them).
func DumpViews(view *NSView, depth int) string {
	root := uintptr(view.Id)
	for s := msg(root, "superview"); s != 0; s = msg(root, "superview") {
		root = s
	}
	var out string
	var walk func(v uintptr, level int)
	walk = func(v uintptr, level int) {
		f := NewNSViewOverload1(int64(v)).Frame()
		mark := ""
		if v == uintptr(view.Id) {
			mark = "  <- content view"
		}
		out += fmt.Sprintf("%*s%s (%.0f,%.0f %.0fx%.0f)%s\n", level*2, "", goString(msg(v, "className")), f.X, f.Y, f.Width, f.Height, mark)
		if level == depth {
			return
		}
		subs := msg(v, "subviews")
		for i, n := uintptr(0), msg(subs, "count"); i < n; i++ {
			walk(msg(subs, "objectAtIndex:", i), level+1)
		}
	}
	walk(root, 0)
	return out
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

// SetGlassCornerRadius rounds the glass view InstallBackdrop added for view; no-op without one.
func SetGlassCornerRadius(view *NSView, r float64) {
	initAssociated()
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	if bd := getAssociated(uintptr(view.Id), backdropKey()); respondsTo(bd, "setCornerRadius:") {
		msgF64(bd, sel("setCornerRadius:"), r)
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
