package swt

import "github.com/haiodo/gowt/internal/gtk"

// NewFontDeviceNameHeightWeight makes a font of a CSS weight (100..900), which is the Pango weight
// number. An empty name means the system font family.
func NewFontDeviceNameHeightWeight(deviceLike DeviceLike, name string, height int32, weight int32, italic bool) *Font {
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
	f.InitNameHeightStyleFontString(name, float32(height), style, nil)
	gtk.OSPango_font_description_set_weight(f.Handle, ClampFontWeight(weight))
	f.impl.init_()
	return f
}
