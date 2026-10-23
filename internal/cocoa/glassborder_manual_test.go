//go:build darwin

package cocoa

import (
	"math"
	"testing"
)

// The stroke is 1/scale points wide and takes separatorColor under the appearance of the view, set
// again by RefreshGlassBorder after the appearance changed; before the glass exists the flag waits.
func TestGlassBorder(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("no NSGlassEffectView")
	}
	onMain(func() {
		setAppearance := func(v uintptr, dark bool) {
			name := "NSAppearanceNameAqua"
			if dark {
				name = "NSAppearanceNameDarkAqua"
			}
			msg(v, "setAppearance:", msg(class("NSAppearance"), "appearanceNamed:", nsString(name)))
		}
		check := func(what string, view *NSView, dark bool) {
			w, c, ok := GlassBorder(view)
			want := SeparatorRGBA(dark)
			if !ok || math.Abs(w-1/backingScale(uintptr(view.Id))) > 1e-9 {
				t.Errorf("%s: border width %v (glass found %v), want 1/scale", what, w, ok)
			}
			for i := range c {
				if math.Abs(c[i]-want[i]) > 0.01 {
					t.Errorf("%s: border colour %v, want separatorColor %v (dark %v)", what, c, want, dark)
					break
				}
			}
		}

		v := msg(msg(class("NSView"), "alloc"), "init")
		view := NewNSViewOverload1(int64(v))
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		SetGlassBorder(view, true)
		InstallPanelBackdrop(view, BackdropGlass)
		SetGlassCornerRadius(view, 18)
		setAppearance(v, false)
		RefreshGlassBorder(view)
		check("panel light", view, false)
		setAppearance(v, true)
		RefreshGlassBorder(view)
		check("panel dark", view, true)
		SetGlassBorder(view, false)
		if w, _, _ := GlassBorder(view); w != 0 {
			t.Errorf("panel: border width %v after off, want 0", w)
		}

		for _, circle := range []bool{false, true} {
			b := msg(msg(class("NSButton"), "alloc"), "init")
			bv := NewNSViewOverload1(int64(b))
			msg(b, "setBezelStyle:", bezelStyleGlass)
			setAppearance(b, true)
			SetGlassBorder(bv, true)
			if circle {
				msg(b, "setBorderShape:", borderShapeCircle)
				ClipGlassCircle(bv)
			}
			check("button dark", bv, true)
		}
	})
}
