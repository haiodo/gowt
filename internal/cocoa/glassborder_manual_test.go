//go:build darwin

package cocoa

import (
	"math"
	"testing"

	"github.com/ebitengine/purego"
)

// The stroke is 1/scale points wide and takes separatorColor under the appearance of the view, set
// again by RefreshGlassBorder after the appearance changed; before the glass exists the flag waits.
func TestGlassBorder(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("no NSGlassEffectView")
	}
	var reduce bool
	onMain(func() { reduce = ReduceTransparency() })
	if reduce {
		t.Skip("Reduce transparency: the panel backdrop is the NSVisualEffectView material, no glass")
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

// The focus ring replaces the separator stroke only while the ring is on and a descendant has focus; the
// flags may come before the glass.
func TestGlassFocusRing(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("no NSGlassEffectView")
	}
	var reduce bool
	onMain(func() { reduce = ReduceTransparency() })
	if reduce {
		t.Skip("Reduce transparency: the panel backdrop is the NSVisualEffectView material, no glass")
	}
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		view := NewNSViewOverload1(int64(v))
		host := msg(msg(class("NSView"), "alloc"), "init")
		msg(host, "addSubview:", v)
		SetGlassBorder(view, true)
		SetGlassFocusRing(view, true)
		SetGlassFocused(view, true)
		InstallPanelBackdrop(view, BackdropGlass)
		SetGlassCornerRadius(view, 18)
		check := func(what string, width float64, rgba [4]float64) {
			w, c, ok := GlassBorder(view)
			if !ok || math.Abs(w-width) > 1e-9 {
				t.Errorf("%s: width %v (glass found %v), want %v", what, w, ok, width)
			}
			for i := range c {
				if math.Abs(c[i]-rgba[i]) > 0.01 {
					t.Errorf("%s: colour %v, want %v", what, c, rgba)
					break
				}
			}
		}
		check("focused", focusRingWidth, FocusRingRGBA(false))
		SetGlassFocused(view, false)
		check("blurred", 1/backingScale(v), SeparatorRGBA(false))
		SetGlassFocused(view, true)
		SetGlassFocusRing(view, false)
		check("ring off", 1/backingScale(v), SeparatorRGBA(false))
	})
}

func TestClippedInspectorGlass(t *testing.T) {
	if !GlassAvailable() {
		t.Skip("no NSGlassEffectView")
	}
	var reduce bool
	onMain(func() { reduce = ReduceTransparency() })
	if reduce {
		t.Skip("Reduce transparency")
	}
	onMain(func() {
		v := msg(msg(class("NSView"), "alloc"), "init")
		host := msg(msg(class("NSView"), "alloc"), "init")
		defer msg(host, "release")
		defer msg(v, "release")
		msg(host, "addSubview:", v)
		view := NewNSViewOverload1(int64(v))
		InstallBackdrop(view, BackdropGlass, false)
		ClipBackdrop(view)
		SetBackdropFrame(view, 0, 0, 260, 500)
		SetGlassCornerRadius(view, 18)
		bd := getAssociated(v, backdropKey())
		glass := msg(msg(bd, "subviews"), "objectAtIndex:", 0)
		if msgToF64 == nil {
			purego.RegisterFunc(&msgToF64, objcMsgSend)
		}
		if radius := msgToF64(glass, sel("cornerRadius")); radius != 18 {
			t.Errorf("radius %v", radius)
		}
		SetGlassCornerRadius(view, 0)
		if radius := msgToF64(glass, sel("cornerRadius")); radius != 0 {
			t.Errorf("flush radius %v", radius)
		}
		SetEdgeLine(view)
		SetEdgeLineHidden(view, true)
		line := getAssociated(v, sel("gowtEdgeLine"))
		if msg(line, "isHidden") == 0 {
			t.Error("floating inspector separator visible")
		}
		SetEdgeLineHidden(view, false)
		if msg(line, "isHidden") != 0 {
			t.Error("flush inspector separator hidden")
		}
	})
}
