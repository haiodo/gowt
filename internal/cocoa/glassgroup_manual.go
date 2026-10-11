//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

var (
	groupClass uintptr
	flipClass  uintptr
	groupKey   = func() uintptr { return sel("gowtGlassGroup") }
	extKey     = func() uintptr { return sel("gowtBackgroundExtension") }
)

// glassGroupContent returns the content view of the NSGlassEffectContainerView that host carries, or 0
// when glass surfaces are not grouped there.
func glassGroupContent(host uintptr, kind int, window bool) uintptr {
	if kind != BackdropGlass || window || !GlassAvailable() {
		return 0
	}
	if g := getAssociated(host, groupKey()); g != 0 {
		return msg(g, "contentView")
	}
	return 0
}

// syncGlassGroup moves the container of host to the frame of host.
func syncGlassGroup(host uintptr) {
	if g := getAssociated(host, groupKey()); g != 0 {
		msgRectOnly(g, sel("setFrame:"), NewNSViewOverload1(int64(host)).Frame())
	}
}

// InstallGlassGroup makes the glass surfaces of the direct children of view share one
// NSGlassEffectContainerView, so surfaces closer than spacing points merge. Call it after SetBackgroundExtension and before the children
// get their glass; it reports false before macOS 26. A second call only changes the spacing.
func InstallGlassGroup(view *NSView, spacing float64) bool {
	initAssociated()
	if !GlassAvailable() || class("NSGlassEffectContainerView") == 0 {
		return false
	}
	v := uintptr(view.Id)
	g := getAssociated(v, groupKey())
	if g == 0 {
		if groupClass == 0 {
			// Transparent to the mouse: the container covers the whole host and would swallow clicks meant for siblings below it.
			noHit := purego.NewCallback(func(self, _ uintptr, x, y float64) uintptr { return 0 })
			groupClass = allocClass(class("NSGlassEffectContainerView"), "GowtGlassGroup", 0)
			addMethod(groupClass, sel("hitTest:"), noHit, "@@:{CGPoint=dd}")
			addMethod(groupClass, sel("isFlipped"), uintptr(OSIsFlipped_CALLBACK()), "c@:")
			registerClass(groupClass)
		}
		g = msg(msg(groupClass, "alloc"), "init")
		// Flipped like the SWT host: backdrop frames are taken from SWT views in that system.
		if flipClass == 0 {
			flipClass = allocClass(class("NSView"), "GowtFlippedView", 0)
			addMethod(flipClass, sel("isFlipped"), uintptr(OSIsFlipped_CALLBACK()), "c@:")
			registerClass(flipClass)
		}
		content := msg(msg(flipClass, "alloc"), "init")
		msg(g, "setContentView:", content)
		msg(content, "release")
		msg(g, "setAutoresizingMask:", viewWidthHeightSizable)
		msgRectOnly(g, sel("setFrame:"), view.Frame())
		// A sibling just below the host, not a child: SWT adds children at the bottom of the host, so a
		// child container would end up above them and cover their content.
		if sv := msg(v, "superview"); sv != 0 {
			msg(sv, "addSubview:positioned:relativeTo:", g, windowBelowArg, v)
		}
		setAssociated(v, groupKey(), g, 1)
		msg(g, "release")
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	if respondsTo(g, "setSpacing:") {
		msgF64(g, sel("setSpacing:"), spacing)
	}
	return true
}

// InstallBackgroundExtension makes an NSBackgroundExtensionView the content view of the window of
// view, with the window's former content view (the SWT view or its scroll view) inside it, so the
// content is extended under a transparent title bar and toolbar. It reports false before macOS 26.
// Off puts the former content view back. Shell.setZOrder_ resets the content view and drops the
// extension, so the association alone is not trusted. Not handled: NO_FOCUS shells compare hitTest
// with the content view, which is the extension view here.
func InstallBackgroundExtension(view *NSView, on bool) bool {
	initAssociated()
	v := uintptr(view.Id)
	w := msg(v, "window")
	if w == 0 || class("NSBackgroundExtensionView") == 0 {
		return false
	}
	ext := getAssociated(v, extKey())
	cur := msg(w, "contentView")
	switch {
	case on && (ext == 0 || cur != ext):
		if ext == 0 {
			ext = msg(msg(class("NSBackgroundExtensionView"), "alloc"), "init")
			setAssociated(v, extKey(), ext, 1)
			msg(ext, "release")
		}
		msg(cur, "retain")
		msgRectOnly(ext, sel("setFrame:"), NewNSViewOverload1(int64(cur)).Frame())
		msg(w, "setContentView:", ext)
		msg(ext, "setContentView:", cur)
		msg(cur, "release")
	case !on && ext != 0:
		inner := cur
		if cur == ext {
			inner = msg(ext, "contentView")
		}
		msg(inner, "retain")
		setAssociated(v, extKey(), 0, 1)
		if cur == ext {
			msg(w, "setContentView:", inner)
		}
		msg(inner, "release")
	}
	return true
}

// BackgroundExtensionView returns the NSBackgroundExtensionView InstallBackgroundExtension made for
// view, 0 if there is none.
func BackgroundExtensionView(view *NSView) uintptr {
	initAssociated()
	return getAssociated(uintptr(view.Id), extKey())
}
