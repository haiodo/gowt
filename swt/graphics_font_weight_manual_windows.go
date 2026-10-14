package swt

// NewFontDeviceNameHeightWeight makes a font with LOGFONT.lfWeight set to a CSS weight (100..900).
// An empty name means the system font face.
func NewFontDeviceNameHeightWeight(deviceLike DeviceLike, name string, height int32, weight int32, italic bool) *Font {
	var device *Device
	if deviceLike != nil {
		device = deviceLike.AsDevice()
	}
	if device == nil {
		device = DeviceGetDevice()
	}
	if device == nil {
		Error(ERROR_NULL_ARGUMENT)
	}
	weight = ClampFontWeight(weight)
	if name == "" {
		name = device.GetSystemFont().GetFontData()[0].GetName()
	}
	var style int32
	if italic {
		style = ITALIC
	}
	fd := NewFontDataOverload3(name, height, style)
	fd.Data.LfWeight = weight
	return NewFontDeviceFd(device, fd)
}
