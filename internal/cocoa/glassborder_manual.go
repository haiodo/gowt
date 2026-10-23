//go:build darwin

// Hand-written: the hairline stroke around a glass panel or a glass button.
package cocoa

import (
	"math"

	"github.com/ebitengine/purego"
)

func borderKey() uintptr { return sel("gowtGlassBorder") }

// SetGlassBorder asks for a one-pixel separatorColor stroke around the glass of a SetGlass panel or a
// glass push button. It can come before the glass: the stroke goes on with the radius or the shape.
func SetGlassBorder(view *NSView, on bool) {
	initAssociated()
	v := uintptr(view.Id)
	flag := uintptr(0)
	if on {
		flag = msg(class("NSNumber"), "numberWithBool:", 1)
	}
	setAssociated(v, borderKey(), flag, 1)
	RefreshGlassBorder(view)
}

// RefreshGlassBorder puts the stroke of SetGlassBorder on the glass layer again: the colour is resolved under
// the view's appearance, and a CALayer keeps the CGColor it was given.
func RefreshGlassBorder(view *NSView) {
	initAssociated()
	v := uintptr(view.Id)
	l := glassBorderLayer(v)
	if l == 0 {
		return
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	if getAssociated(v, borderKey()) == 0 {
		msgF64(l, sel("setBorderWidth:"), 0)
		return
	}
	msgF64(l, sel("setBorderWidth:"), 1/backingScale(v))
	msg(l, "setBorderColor:", separatorCGColor(v))
}

// glassBorderLayer is the layer that carries the glass shape: the glass view of a panel, the button itself
// for a glass bezel. 0 while there is none.
func glassBorderLayer(v uintptr) uintptr {
	if bd := getAssociated(v, backdropKey()); bd != 0 && panelGlassInside(v, bd) {
		if l := msg(bd, "layer"); respondsTo(l, "setBorderWidth:") {
			return l
		}
		return 0
	}
	if !respondsTo(v, "bezelStyle") || msg(v, "bezelStyle") != bezelStyleGlass {
		return 0
	}
	msg(v, "setWantsLayer:", 1)
	l := msg(v, "layer")
	if !respondsTo(l, "setBorderWidth:") {
		return 0
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	// a glass bezel that is not a circle is a capsule; the layer is not clipped, only the stroke follows it
	if ButtonBorderShape(NewNSViewOverload1(int64(v))) != borderShapeCircle {
		f := NewNSViewOverload1(int64(v)).Frame()
		msgF64(l, sel("setCornerRadius:"), math.Min(f.Width, f.Height)/2)
	}
	return l
}

// backingScale is the pixels per point of the window of v, the main screen's before v is in a window.
func backingScale(v uintptr) float64 {
	if msgToF64 == nil {
		purego.RegisterFunc(&msgToF64, objcMsgSend)
	}
	holder := msg(v, "window")
	if holder == 0 {
		holder = msg(class("NSScreen"), "mainScreen")
	}
	if s := msgToF64(holder, sel("backingScaleFactor")); s >= 1 {
		return s
	}
	return 1
}

// BackingScale is the pixels per point the stroke of view is measured in, for tests.
func BackingScale(view *NSView) float64 { return backingScale(uintptr(view.Id)) }

// separatorCGColor is NSColor.separatorColor resolved under the appearance of v.
func separatorCGColor(v uintptr) uintptr {
	ap := class("NSAppearance")
	old := msg(ap, "currentAppearance")
	msg(old, "retain")
	msg(ap, "setCurrentAppearance:", msg(v, "effectiveAppearance"))
	cg := msg(msg(class("NSColor"), "separatorColor"), "CGColor")
	msg(ap, "setCurrentAppearance:", old)
	msg(old, "release")
	return cg
}

// GlassBorder reads the stroke back for tests: width in points and the colour as sRGB components; ok is
// false without a glass layer.
func GlassBorder(view *NSView) (width float64, rgba [4]float64, ok bool) {
	initAssociated()
	l := glassBorderLayer(uintptr(view.Id))
	if l == 0 {
		return 0, rgba, false
	}
	if msgToF64 == nil {
		purego.RegisterFunc(&msgToF64, objcMsgSend)
	}
	width = msgToF64(l, sel("borderWidth"))
	return width, cgColorRGBA(msg(l, "borderColor")), true
}

// SeparatorRGBA is NSColor.separatorColor under the Aqua or Dark Aqua appearance, for tests.
func SeparatorRGBA(dark bool) [4]float64 {
	name := "NSAppearanceNameAqua"
	if dark {
		name = "NSAppearanceNameDarkAqua"
	}
	ap := class("NSAppearance")
	old := msg(ap, "currentAppearance")
	msg(old, "retain")
	msg(ap, "setCurrentAppearance:", msg(ap, "appearanceNamed:", nsString(name)))
	c := cgColorRGBA(msg(msg(class("NSColor"), "separatorColor"), "CGColor"))
	msg(ap, "setCurrentAppearance:", old)
	msg(old, "release")
	return c
}

func cgColorRGBA(cg uintptr) (rgba [4]float64) {
	if cg == 0 {
		return rgba
	}
	c := msg(msg(class("NSColor"), "colorWithCGColor:", cg), "colorUsingColorSpace:", msg(class("NSColorSpace"), "sRGBColorSpace"))
	if c == 0 {
		return rgba
	}
	n := NewNSColorOverload1(int64(c))
	return [4]float64{n.RedComponent(), n.GreenComponent(), n.BlueComponent(), n.AlphaComponent()}
}
