//go:build darwin

package cocoa

import (
	"testing"

	"github.com/ebitengine/purego"
)

const phW, phH = 240, 80

// placeholderInk draws the placeholder of tv into a flipped 1x bitmap, as the text view's drawRect
// would, and returns the box of the drawn pixels and their highest alpha (ok false: nothing drawn).
func placeholderInk(tv uintptr, s string) (minX, minY, maxY int, alpha float64, ok bool) {
	var translate, scale func(cg uintptr, x, y float64)
	purego.RegisterLibFunc(&translate, purego.RTLD_DEFAULT, "CGContextTranslateCTM")
	purego.RegisterLibFunc(&scale, purego.RTLD_DEFAULT, "CGContextScaleCTM")
	rep := NewNSBitmapImageRep()
	rep.Id = int64(msg(class("NSBitmapImageRep"), "alloc"))
	rep = rep.InitWithBitmapDataPlanes(0, phW, phH, 8, 4, true, false, OSNSDeviceRGBColorSpace_, 0, 4*phW, 32)
	defer rep.Release()
	NSGraphicsContextStatic_saveGraphicsState()
	cg := msg(uintptr(NSGraphicsContextGraphicsContextWithBitmapImageRep(rep).Id), "CGContext")
	translate(cg, 0, phH)
	scale(cg, 1, -1)
	msg(class("NSGraphicsContext"), "setCurrentContext:", msg(class("NSGraphicsContext"), "graphicsContextWithCGContext:flipped:", cg, 1))
	DrawTextViewPlaceholder(NewNSViewOverload1(int64(tv)), s)
	NSGraphicsContextStatic_restoreGraphicsState()
	minX, minY = phW, phH
	for y := 0; y < phH; y++ {
		for x := 0; x < phW; x++ {
			a := NewNSColorOverload1(int64(msg(uintptr(rep.Id), "colorAtX:y:", uintptr(x), uintptr(y)))).AlphaComponent()
			if a < 0.05 {
				continue
			}
			ok = true
			minX, minY, maxY = min(minX, x), min(minY, y), max(maxY, y)
			alpha = max(alpha, a)
		}
	}
	return minX, minY, maxY, alpha, ok
}

// No Display and no window: the placeholder starts at the text container origin plus the line
// fragment padding, in the view's font and a translucent placeholder colour, and only while empty.
func TestDrawTextViewPlaceholder(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		tv := msg(msg(class("NSTextView"), "alloc"), "init")
		defer msg(tv, "release")
		msgRectOnly(tv, sel("setFrame:"), NSRect{Width: phW, Height: phH})
		var setInset func(self, sel uintptr, s NSSize)
		purego.RegisterFunc(&setInset, objcMsgSend)
		setInset(tv, sel("setTextContainerInset:"), NSSize{Width: 7, Height: 9})
		var systemFont func(cls, sel uintptr, size float64) uintptr
		purego.RegisterFunc(&systemFont, objcMsgSend)
		msg(tv, "setFont:", systemFont(class("NSFont"), sel("systemFontOfSize:"), 13))
		x, y, bottom, alpha, ok := placeholderInk(tv, "Hello")
		if !ok {
			fatalf(t, "nothing drawn for an empty view")
		}
		// "H" has a stem at its left side bearing of about one point; the inset is 7 and the padding 5.
		if x < 12 || x > 14 || y < 9 || y > 14 {
			t.Errorf("ink starts at %d,%d, want about 12,10: inset 7,9 plus padding 5", x, y)
		}
		if alpha > 0.6 {
			t.Errorf("ink alpha %.2f, want the translucent placeholderTextColor", alpha)
		}
		small := bottom - y

		msg(tv, "setFont:", systemFont(class("NSFont"), sel("systemFontOfSize:"), 26))
		if _, y2, bottom2, _, _ := placeholderInk(tv, "Hello"); bottom2-y2 < 2*small-2 {
			t.Errorf("26 pt ink is %d px high, 13 pt %d: the hint does not follow the font", bottom2-y2, small)
		}

		msg(tv, "setString:", nsString("x"))
		if _, _, _, _, ok := placeholderInk(tv, "Hello"); ok {
			t.Error("drawn while the view holds text")
		}
	})
}
