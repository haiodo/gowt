//go:build darwin

package cocoa

import (
	"math"
	"testing"
)

// A borderless toggle draws the same on and off unless SyncButtonTint gives it the accent colour while
// on; drawn off screen, the symbol of the tinted one differs in colour from the plain one.
func TestBorderlessToggleTint(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		img := msg(class("NSImage"), "imageWithSystemSymbolName:accessibilityDescription:", nsString("eye"), 0)
		if img == 0 {
			t.Skip("no SF Symbols")
		}
		render := func(on bool) (r, g, b float64) {
			v := msg(msg(class("NSButton"), "alloc"), "init")
			defer msg(v, "release")
			msg(v, "setButtonType:", 1) // NSButtonTypePushOnPushOff, as SWT's TOGGLE
			view := NewNSViewOverload1(int64(v))
			SetButtonBorderless(view)
			msg(v, "setTitle:", nsString(""))
			msg(v, "setImage:", img)
			msg(v, "setState:", boolArg(on))
			SyncButtonTint(view)
			if ButtonTinted(view) != on {
				t.Errorf("on %v: tinted %v", on, ButtonTinted(view))
			}
			rect := NSRect{Width: 28, Height: 22}
			msgRectOnly(v, sel("setFrame:"), rect)
			rep := view.BitmapImageRepForCachingDisplayInRect(rect)
			view.CacheDisplayInRect(rect, rep)
			w, h := int(msg(uintptr(rep.Id), "pixelsWide")), int(msg(uintptr(rep.Id), "pixelsHigh"))
			sum := 0.0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c := NewNSColorOverload1(int64(msg(uintptr(rep.Id), "colorAtX:y:", uintptr(x), uintptr(y))))
					a := c.AlphaComponent()
					r, g, b = r+a*c.RedComponent(), g+a*c.GreenComponent(), b+a*c.BlueComponent()
					sum += a
				}
			}
			if sum == 0 {
				t.Fatalf("on %v: nothing drawn", on)
			}
			return r / sum, g / sum, b / sum
		}
		r0, g0, b0 := render(false)
		r1, g1, b1 := render(true)
		if d := max(math.Abs(r1-r0), math.Abs(g1-g0), math.Abs(b1-b0)); d < 0.1 {
			t.Errorf("on (%.2f %.2f %.2f) looks like off (%.2f %.2f %.2f)", r1, g1, b1, r0, g0, b0)
		}
	})
}
