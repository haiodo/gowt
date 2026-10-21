//go:build darwin

// Hand-written: round and borderless NSButtons, accessibility label.
package cocoa

const (
	bezelStyleCircular = 7
	borderShapeCircle  = 3 // NSControlBorderShapeCircle, macOS 26
)

// SetButtonBorderless removes the bezel. AppKit still dims the image while the button is pressed.
func SetButtonBorderless(view *NSView) {
	msg(uintptr(view.Id), "setBordered:", 0)
}

// ButtonBordered reports whether the button draws a bezel.
func ButtonBordered(view *NSView) bool {
	return msg(uintptr(view.Id), "isBordered")&0xff != 0
}

// SetButtonCircle makes the bezel round: the circular bezel before macOS 26, and the circle border
// shape on 26 and newer. A glass bezel (SetGlassButton) stays and only gets the shape.
func SetButtonCircle(view *NSView) {
	v := uintptr(view.Id)
	if msg(v, "bezelStyle") != bezelStyleGlass {
		msg(v, "setBezelStyle:", bezelStyleCircular)
	}
	if respondsTo(v, "setBorderShape:") {
		msg(v, "setBorderShape:", borderShapeCircle)
	}
}

// ButtonBezelStyle returns the NSBezelStyle of the button.
func ButtonBezelStyle(view *NSView) int { return int(msg(uintptr(view.Id), "bezelStyle")) }

// ButtonBorderShape returns the NSControlBorderShape, 0 before macOS 26.
func ButtonBorderShape(view *NSView) int {
	v := uintptr(view.Id)
	if !respondsTo(v, "borderShape") {
		return 0
	}
	return int(msg(v, "borderShape"))
}

// SetAccessibilityLabel is what VoiceOver reads for a control without text, such as an icon button.
func SetAccessibilityLabel(view *NSView, s string) {
	msg(uintptr(view.Id), "setAccessibilityLabel:", nsString(s))
}

// AccessibilityLabel returns the label set by SetAccessibilityLabel.
func AccessibilityLabel(view *NSView) string {
	return goString(msg(uintptr(view.Id), "accessibilityLabel"))
}

// SetGlassButton gives a push NSButton the macOS 26 glass bezel; older systems and borderless
// buttons keep what they have. AppKit draws that bezel only for a view class without drawRect: and
// a plain NSButtonCell, so the SWT button moves to SWTGlassButton and a plain cell first.
func SetGlassButton(view *NSView) {
	v := uintptr(view.Id)
	if !GlassAvailable() || !respondsTo(v, "setBezelStyle:") || !ButtonBordered(view) {
		return
	}
	round := msg(v, "bezelStyle") == bezelStyleCircular
	if cls := class("SWTGlassButton"); cls != 0 && uintptr(OSObject_getClass(int64(v))) != cls {
		moveToPlainCell(v, cls)
	}
	msg(v, "setBezelStyle:", bezelStyleGlass)
	if round {
		SetButtonCircle(view)
	}
}

// moveToPlainCell copies what SWT put on the button's cell to a new NSButtonCell, so the SWT cell
// subclass and the SWT view class stop overriding the drawing.
func moveToPlainCell(v, cls uintptr) {
	old := msg(v, "cell")
	c := msg(msg(class("NSButtonCell"), "alloc"), "init")
	msg(c, "setButtonType:", 0)
	msg(c, "setAttributedTitle:", msg(old, "attributedTitle"))
	msg(c, "setImage:", msg(old, "image"))
	msg(c, "setImagePosition:", msg(old, "imagePosition"))
	msg(c, "setAlignment:", msg(old, "alignment"))
	msg(c, "setFont:", msg(old, "font"))
	msg(c, "setControlSize:", msg(old, "controlSize"))
	msg(c, "setKeyEquivalent:", msg(old, "keyEquivalent"))
	msg(c, "setTarget:", msg(old, "target"))
	msg(c, "setAction:", msg(old, "action"))
	msg(c, "setEnabled:", msg(old, "isEnabled")&0xff)
	if w := msg(v, "window"); w != 0 && msg(w, "defaultButtonCell") == old {
		msg(w, "setDefaultButtonCell:", c)
	}
	OSObject_setClass(int64(v), int64(cls))
	msg(v, "setCell:", c)
	msg(c, "release")
}
