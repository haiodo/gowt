package main

import "github.com/ebitengine/purego/objc"

// systemAccent is the user's accent colour from System Settings, converted to sRGB. UI thread only.
func systemAccent() (r, g, b uint8, ok bool) {
	nsColor := objc.ID(objc.GetClass("NSColor"))
	c := nsColor.Send(objc.RegisterName("controlAccentColor"))
	space := objc.ID(objc.GetClass("NSColorSpace")).Send(objc.RegisterName("sRGBColorSpace"))
	c = c.Send(objc.RegisterName("colorUsingColorSpace:"), space)
	if c == 0 {
		return 0, 0, 0, false
	}
	comp := func(name string) uint8 {
		v := objc.Send[float64](c, objc.RegisterName(name))
		return uint8(min(max(v, 0), 1)*255 + 0.5)
	}
	return comp("redComponent"), comp("greenComponent"), comp("blueComponent"), true
}
