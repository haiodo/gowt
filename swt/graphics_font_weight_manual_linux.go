package swt

import "github.com/haiodo/gowt/internal/gtk"

// NewFontDeviceNameHeightWeight makes a font of a CSS weight (100..900), which is the Pango weight
// number. An empty name means the system font family.
func NewFontDeviceNameHeightWeight(deviceLike DeviceLike, name string, height int32, weight int32, italic bool) *Font {
	return newFontWeight(deviceLike, name, float32(height), weight, italic)
}

// NewFontDevicePx makes a font px logical pixels tall: Pango sizes are points at the screen resolution.
func NewFontDevicePx(deviceLike DeviceLike, name string, px float32, weight int32, italic bool) *Font {
	var device *Device
	if deviceLike != nil {
		device = deviceLike.AsDevice()
	}
	if device == nil {
		device = DeviceGetDevice()
	}
	return newFontWeight(device, name, px*72/float32(device.GetScreenDPI().Y), weight, italic)
}

func newFontWeight(deviceLike DeviceLike, name string, height float32, weight int32, italic bool) *Font {
	var device *Device
	if deviceLike != nil {
		device = deviceLike.AsDevice()
	}
	f := newFont(device)
	if name == "" {
		name = f.device.GetSystemFont().GetFontData()[0].GetName()
	}
	var style int32
	if italic {
		style = ITALIC
	}
	f.InitNameHeightStyleFontString(name, height, style, nil)
	gtk.OSPango_font_description_set_weight(f.Handle, ClampFontWeight(weight))
	f.impl.init_()
	return f
}
