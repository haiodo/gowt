//go:build darwin

// Hand-written: round and borderless NSButtons, accessibility label.
package cocoa

import (
	"math"

	"github.com/ebitengine/purego"
)

const (
	bezelStyleCircular = 7
	bezelStyleRecessed = 13 // NSBezelStyleAccessoryBar, the scope-bar button of Finder's search
	borderShapeCircle  = 3  // NSControlBorderShapeCircle, macOS 26
)

// SetButtonBorderless removes the bezel. AppKit still dims the image while the button is pressed.
func SetButtonBorderless(view *NSView) {
	msg(uintptr(view.Id), "setBordered:", 0)
}

// SetButtonRecessed gives a toggle button the recessed bezel, drawn only under the pointer while the
// button is off; on, it shows the pressed-in fill.
func SetButtonRecessed(view *NSView) {
	v := uintptr(view.Id)
	msg(v, "setBezelStyle:", bezelStyleRecessed)
	msg(v, "setShowsBorderOnlyWhileMouseInside:", 1)
}

// SetControlSmall gives a control NSControlSizeSmall and the system font of that size.
func SetControlSmall(view *NSView) { setSmall(uintptr(view.Id)) }

// ControlSize returns the NSControlSize of a control.
func ControlSize(view *NSView) int { return int(msg(uintptr(view.Id), "controlSize")) }

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
	ClipGlassCircle(view)
}

// ClipGlassCircle clips a glass circle button to its circle. On macOS 26.5 the glass view draws a
// square, a little darker than the background, over the whole frame behind the circle; macOS 27 does
// not. Call it again after the button is resized.
func ClipGlassCircle(view *NSView) {
	defer RefreshGlassBorder(view)
	v := uintptr(view.Id)
	if msg(v, "bezelStyle") != bezelStyleGlass || ButtonBorderShape(view) != borderShapeCircle {
		return
	}
	initAssociated()
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	f := view.Frame()
	msg(v, "setWantsLayer:", 1)
	l := msg(v, "layer")
	msgF64(l, sel("setCornerRadius:"), math.Min(f.Width, f.Height)/2)
	msg(l, "setMasksToBounds:", 1)
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
	RefreshGlassBorder(view)
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

// SetButtonDestructive marks the button's action as destructive (NSButton.hasDestructiveAction, macOS 11).
func SetButtonDestructive(view *NSView) {
	if v := uintptr(view.Id); respondsTo(v, "setHasDestructiveAction:") {
		msg(v, "setHasDestructiveAction:", 1)
	}
}

// ButtonDestructive reports hasDestructiveAction.
func ButtonDestructive(view *NSView) bool {
	v := uintptr(view.Id)
	return respondsTo(v, "hasDestructiveAction") && msg(v, "hasDestructiveAction")&0xff != 0
}
