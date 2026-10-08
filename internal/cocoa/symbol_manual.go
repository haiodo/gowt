//go:build darwin

package cocoa

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	symbolOnce sync.Once
	// objc_msgSend with float and struct arguments needs a typed signature; msg only passes words.
	msgSymbolConfig func(self, sel uintptr, pointSize, weight float64, scale int64) uintptr
	msgSize         func(self, sel uintptr) NSSize
	msgDrawInRect   func(self, sel uintptr, dst, src NSRect, op uint64, fraction float64)
)

// SymbolAlpha draws the SF Symbol name into a px x px box (aspect kept, centered) and returns its
// alpha channel row by row; nil when the symbol does not exist or the OS predates SF Symbols (macOS 11).
// Call on the UI thread. Only alpha is returned so the caller tints the template as it likes.
func SymbolAlpha(name string, px int) []uint8 {
	objcInit()
	symbolOnce.Do(func() {
		purego.RegisterFunc(&msgSymbolConfig, objcMsgSend)
		purego.RegisterFunc(&msgSize, objcMsgSend)
		purego.RegisterFunc(&msgDrawInRect, objcMsgSend)
	})
	nsImage := class("NSImage")
	if msg(nsImage, "respondsToSelector:", sel("imageWithSystemSymbolName:accessibilityDescription:")) == 0 {
		return nil
	}
	pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
	defer msg(pool, "drain")
	img := msg(nsImage, "imageWithSystemSymbolName:accessibilityDescription:", nsString(name), 0)
	if img == 0 {
		return nil
	}
	// pointSize does not matter: the template is a vector scaled by drawInRect. Weight 0 is NSFontWeightRegular,
	// scale 2 is NSImageSymbolScaleMedium.
	cfg := msgSymbolConfig(class("NSImageSymbolConfiguration"), sel("configurationWithPointSize:weight:scale:"), float64(px), 0, 2)
	if cfg != 0 {
		if configured := msg(img, "imageWithSymbolConfiguration:", cfg); configured != 0 {
			img = configured
		}
	}
	size := msgSize(img, sel("size"))
	if size.Width <= 0 || size.Height <= 0 {
		return nil
	}
	rep := msg(msg(class("NSBitmapImageRep"), "alloc"), "initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:",
		0, uintptr(px), uintptr(px), 8, 4, 1, 0, nsString("NSDeviceRGBColorSpace"), 0, 0)
	if rep == 0 {
		return nil
	}
	defer msg(rep, "release")
	data := msg(rep, "bitmapData")
	stride := int(msg(rep, "bytesPerRow"))
	if data == 0 || stride < px*4 {
		return nil
	}
	pix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), data)), stride*px)
	clear(pix)

	gc := class("NSGraphicsContext")
	ctx := msg(gc, "graphicsContextWithBitmapImageRep:", rep)
	if ctx == 0 {
		return nil
	}
	msg(gc, "saveGraphicsState")
	msg(gc, "setCurrentContext:", ctx)
	k := min(float64(px)/size.Width, float64(px)/size.Height)
	w, h := size.Width*k, size.Height*k
	msgDrawInRect(img, sel("drawInRect:fromRect:operation:fraction:"),
		NSRect{(float64(px) - w) / 2, (float64(px) - h) / 2, w, h}, NSRect{}, 2, 1)
	msg(ctx, "flushGraphics")
	msg(gc, "restoreGraphicsState")

	alpha := make([]uint8, px*px)
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			alpha[y*px+x] = pix[y*stride+x*4+3]
		}
	}
	return alpha
}
