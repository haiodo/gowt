//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

var msg4F64 func(id, sel uintptr, a, b, c, d float64) uintptr

// SetFieldFill gives a single-line field a rounded fill behind its text: the bezel and the field's
// own background go, a layer takes the background colour already set on the field and the radius.
func SetFieldFill(view *NSView, radius float64) {
	v := uintptr(view.Id)
	msg(v, "setBezeled:", 0)
	msg(v, "setBordered:", 0)
	msg(v, "setDrawsBackground:", 0)
	if e := msg(v, "currentEditor"); e != 0 {
		msg(e, "setDrawsBackground:", 0)
	}
	msg(v, "setWantsLayer:", 1)
	l := msg(v, "layer")
	if l == 0 {
		return
	}
	if c := msg(v, "backgroundColor"); c != 0 {
		msg(l, "setBackgroundColor:", msg(c, "CGColor"))
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	msgF64(l, sel("setCornerRadius:"), radius)
	msg(l, "setMasksToBounds:", 1)
	msg(v, "setNeedsDisplay:", 1)
}

// SetFieldStroke draws a border of the RGBA colour around the field's fill, width in points.
func SetFieldStroke(view *NSView, red, green, blue, alpha, width float64) {
	v := uintptr(view.Id)
	msg(v, "setWantsLayer:", 1)
	l := msg(v, "layer")
	if l == 0 {
		return
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	if msg4F64 == nil {
		purego.RegisterFunc(&msg4F64, objcMsgSend)
	}
	c := msg4F64(class("NSColor"), sel("colorWithSRGBRed:green:blue:alpha:"), red, green, blue, alpha)
	msg(l, "setBorderColor:", msg(c, "CGColor"))
	msgF64(l, sel("setBorderWidth:"), width)
	msg(v, "setNeedsDisplay:", 1)
}

// SetFieldInset puts inset points between the edge of a single-line field and its text, left and right.
func SetFieldInset(view *NSView, inset float64) {
	affixInit()
	v := uintptr(view.Id)
	st := affixStateOf(v)
	st.Inset = inset
	msg(v, "invalidateIntrinsicContentSize")
	msg(v, "setNeedsDisplay:", 1)
}

// FieldFillRadius is the corner radius of the layer SetFieldFill added, 0 without one.
func FieldFillRadius(view *NSView) float64 {
	l := msg(uintptr(view.Id), "layer")
	if l == 0 || !respondsTo(l, "cornerRadius") {
		return 0
	}
	if msgToF64 == nil {
		purego.RegisterFunc(&msgToF64, objcMsgSend)
	}
	return msgToF64(l, sel("cornerRadius"))
}

// FieldFillRGBA is the fill of SetFieldFill as sRGB; ok is false without a layer background.
func FieldFillRGBA(view *NSView) ([4]float64, bool) {
	l := msg(uintptr(view.Id), "layer")
	if l == 0 {
		return [4]float64{}, false
	}
	c := msg(l, "backgroundColor")
	if c == 0 {
		return [4]float64{}, false
	}
	return cgColorRGBA(c), true
}

// FieldInset is the inset SetFieldInset gave the field, 0 without one.
func FieldInset(view *NSView) float64 {
	if st := affixes[uintptr(view.Id)]; st != nil {
		return st.Inset
	}
	return 0
}
