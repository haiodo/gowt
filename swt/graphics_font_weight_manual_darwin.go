package swt

import "github.com/haiodo/gowt/internal/cocoa"

// NewFontDeviceNameHeightWeight makes a font of a CSS weight (100..900). An empty name or the system
// family gives the system UI font at that weight; an unknown family falls back to it.
func NewFontDeviceNameHeightWeight(deviceLike DeviceLike, name string, height int32, weight int32, italic bool) *Font {
	return newFontWeight(deviceLike, name, float32(height), weight, italic)
}

// NewFontDevicePx makes a font px logical pixels tall, which on macOS is the point size.
func NewFontDevicePx(deviceLike DeviceLike, name string, px float32, weight int32, italic bool) *Font {
	return newFontWeight(deviceLike, name, px, weight, italic)
}

func newFontWeight(deviceLike DeviceLike, name string, height float32, weight int32, italic bool) *Font {
	var device *Device
	if deviceLike != nil {
		device = deviceLike.AsDevice()
	}
	weight = ClampFontWeight(weight)
	var pool *cocoa.NSAutoreleasePool = nil
	if !cocoa.NSThreadIsMainThread() {
		pool = castcocoaNSObjectTococoaNSAutoreleasePool(cocoa.NewNSAutoreleasePool().Alloc().Init())
	}
	defer func() {
		if pool != nil {
			pool.Release()
		}
	}()
	f := newFont(device)
	dpi := f.device.dpi
	screenDPI := f.device.GetScreenDPI()
	size := float64(height * float32(dpi.Y) / float32(screenDPI.Y))
	system := cocoa.NSFontSystemFontOfSizeWeight(size, weight)
	if name == "" || system.FamilyName().GetString() == name {
		f.Handle = system
	} else {
		f.Handle = cocoa.NSFontWithFamilyWeight(name, weight, size)
	}
	var style int32
	if italic {
		style = ITALIC
	}
	f.InitTraits(style, system)
	f.Handle.Retain()
	f.impl.init_()
	return f
}
